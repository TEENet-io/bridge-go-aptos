package ethtxmanager

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/TEENet-io/bridge-go/state"
	"github.com/btcsuite/btcd/chaincfg"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	// eth synchronizer config
	frequencyToCheckEthFinalizedBlock = 100 * time.Millisecond

	// eth tx manager config
	frequencyToPrepareRedeem      = 500 * time.Millisecond
	frequencyToMint               = 500 * time.Millisecond // 0.5 second
	frequencyToMonitorPendingTxs  = 500 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 10

	// blockInterval = 100 * time.Millisecond
)

// Gather test tools on Ethereum side.
type testEnv struct {
	sim *etherman.SimEtherman

	sqldb   *sql.DB
	statedb *state.StateDB
	st      *state.State
	mgrdb   *EthTxManagerDB
	mgr     *EthTxManager
	sync    *ethsync.Synchronizer
}

func newTestEnv(t *testing.T, file string, btcChainConfig *chaincfg.Params) *testEnv {

	ss, err := multisig_client.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("failed to create schnorr wallet: %v", err)
	}
	sim, err := etherman.NewSimEtherman(etherman.GenPrivateKeys(10), ss, big.NewInt(1337))
	assert.NoError(t, err)

	chainID, err := sim.Etherman.Client().ChainID(context.Background())
	assert.NoError(t, err)

	// create a sql db
	sqldb, err := sql.Open("sqlite3", file)
	assert.NoError(t, err)

	// create a eth2btc state db
	statedb, err := state.NewStateDB(sqldb)
	assert.NoError(t, err)

	// create a eth2btc state from the eth2btc statedb
	st, err := state.New(statedb, &state.StateConfig{ChannelSize: 1})
	assert.NoError(t, err)

	// create a eth tx manager db
	mgrdb, err := NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)

	// create a eth synchronizer
	sync, err := ethsync.New(
		sim.Etherman,
		st,
		&ethsync.EthSyncConfig{
			IntervalCheckBlockchain: frequencyToCheckEthFinalizedBlock,
			// TODO, can be other network params.
			BtcChainConfig: btcChainConfig,
			EthChainID:     chainID,
		},
	)
	assert.NoError(t, err)

	// create a eth tx manager
	cfg := &EthTxMgrConfig{
		IntervalToPrepareRedeem:       frequencyToPrepareRedeem,
		IntervalToMint:                frequencyToMint,
		IntervalToMonitorPendingTxs:   frequencyToMonitorPendingTxs,
		TimeoutOnWaitingForSignature:  timeoutOnWaitingForSignature,
		TimeoutOnWaitingForOutpoints:  timtoutOnWaitingForOutpoints,
		TimeoutOnMonitoringPendingTxs: timeoutOnMonitoringPendingTxs,
	}
	// TODO use our btc wallet instead
	btcWallet := &MockBtcWallet{}

	schnorrWallet, _ := NewRandomMockedSchnorrAsyncSigner()
	mgr, err := NewEthTxManager(cfg, sim.Etherman, statedb, mgrdb, schnorrWallet, btcWallet)
	assert.NoError(t, err)

	return &testEnv{sim, sqldb, statedb, st, mgrdb, mgr, sync}
}

func (env *testEnv) close() {
	env.mgrdb.Close()
	env.statedb.Close()
	env.sqldb.Close()
}

func randFile() string {
	return ethcommon.Hash(common.RandBytes32()).String() + ".db"
}

// 1) simulate a mint() Tx on EVM
// 2) simulate/insert a Mint Event in statedb (should be created by btc side).
// It checks if the Mint is captured and monitored by the manager.
func TestOnIsMinted(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	// Test is done on mainnet of bitcoin
	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	// simulate a mint() on EVM chain
	// btctxid is random
	_, params := env.sim.Mint(common.RandBytes32(), 1, 100)
	commit()

	// Also simulate that BTC side has
	// Inserted mint event to the database.
	mint := &state.Mint{
		BtcTxId:  params.BtcTxId,
		Receiver: params.Receiver,
		Amount:   common.BigIntClone(params.Amount),
	}
	err := env.statedb.InsertMint(mint)
	assert.NoError(t, err)

	// Start manager, to monitor the minted tx on EVM
	// 1) gather from statedb about suppose to be minted evm tx.
	// 2) track the tx.
	// 2) update mgrdb about sucessful minted evm tx.
	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToMint * 2)
	cancel()

	// Check if the minted tx is mined and mgrdb captured it.
	mts, err := env.mgrdb.GetMonitoredTxsById(mint.BtcTxId)
	assert.NoError(t, err)
	assert.Len(t, mts, 0)
}

// TestOnCheckBeforeMint tests the checks before entering the mint process
func TestOnCheckBeforeMint(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()

	unMinted := state.RandMint(false)
	err := env.statedb.InsertMint(unMinted)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToMint * 2)
	cancel()
	mts, err := env.mgrdb.GetMonitoredTxsById(unMinted.BtcTxId)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	assert.Equal(t, common.EmptyHash, mts[0].MinedAt)
	env.mgrdb.DeleteMonitoredTxByTxHash(mts[0].TxHash)

	// prepare a redeem when there are associated monitored tx in the table.
	// Not all the tx are with status Timeout, so no new monitored tx should be added
	mt1 := RandMonitoredTx(Pending, 1)
	mt1.Id = unMinted.BtcTxId
	mt2 := RandMonitoredTx(Timeout, 1)
	mt2.Id = unMinted.BtcTxId
	err = env.mgrdb.InsertMonitoredTx(mt1)
	assert.NoError(t, err)
	err = env.mgrdb.InsertMonitoredTx(mt2)
	assert.NoError(t, err)
	ctx, cancel = context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()
	mts, err = env.mgrdb.GetMonitoredTxsById(unMinted.BtcTxId)
	assert.NoError(t, err)
	assert.Len(t, mts, 2)
	err = env.mgrdb.DeleteMonitoredTxByTxHash(mt1.TxHash)
	assert.NoError(t, err)
	err = env.mgrdb.DeleteMonitoredTxByTxHash(mt2.TxHash)
	assert.NoError(t, err)

	// prepare a redeem when there is an associated monitored tx in the table.
	// The status of the tx is Timeout, so a new monitored tx should be added
	mt := RandMonitoredTx(Timeout, 1)
	mt.Id = unMinted.BtcTxId
	err = env.mgrdb.InsertMonitoredTx(mt)
	assert.NoError(t, err)
	ctx, cancel = context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()
	mts, err = env.mgrdb.GetMonitoredTxsById(unMinted.BtcTxId)
	assert.NoError(t, err)
	assert.Len(t, mts, 2)
	for _, m := range mts {
		if m.Status == Timeout {
			assert.Equal(t, mt, m)
		} else {
			assert.Equal(t, unMinted.BtcTxId, m.Id)
		}
	}
}

// TestIsPrepared tests the case where the redeem is already prepared on chain
func TestIsPrepared(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	// prepare the requestTxHash on chain
	_, params := env.sim.Prepare(1, 100, 1, 1)
	commit()

	// insert a requested redeem with the same requestTxHash to the state db
	redeem := &state.Redeem{
		RequestTxHash: params.RequestTxHash,
		Requester:     params.Requester,
		Receiver:      params.Receiver,
		Amount:        common.BigIntClone(params.Amount),
		Status:        state.RedeemStatusRequested,
	}
	err := env.statedb.InsertAfterRequested(redeem)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() { err = env.mgr.Start(ctx) }()

	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()

	// no new tx created for monitoring
	mtxs, err := env.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Len(t, mtxs, 0)
}

// TestOnCheckBeforePrepare tests the checks before entering the prepare process
func TestOnCheckBeforePrepare(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()

	redeem := state.RandRedeem(state.RedeemStatusRequested)
	err := env.statedb.InsertAfterRequested(redeem)
	assert.NoError(t, err)

	// prepare a redeem when no associated monitored tx in the table
	// expected to find a new monitored tx added to the table
	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()
	mts, err := env.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	assert.Equal(t, redeem.RequestTxHash, mts[0].Id)
	err = env.mgrdb.DeleteMonitoredTxByTxHash(mts[0].TxHash)
	assert.NoError(t, err)

	// prepare a redeem when there are associated monitored tx in the table.
	// Not all the tx are with status Timeout, so no new monitored tx should be added
	mt1 := RandMonitoredTx(Pending, 1)
	mt1.Id = redeem.RequestTxHash
	mt2 := RandMonitoredTx(Timeout, 1)
	mt2.Id = redeem.RequestTxHash
	err = env.mgrdb.InsertMonitoredTx(mt1)
	assert.NoError(t, err)
	err = env.mgrdb.InsertMonitoredTx(mt2)
	assert.NoError(t, err)
	ctx, cancel = context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()
	mts, err = env.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Len(t, mts, 2)
	err = env.mgrdb.DeleteMonitoredTxByTxHash(mt1.TxHash)
	assert.NoError(t, err)
	err = env.mgrdb.DeleteMonitoredTxByTxHash(mt2.TxHash)
	assert.NoError(t, err)

	// prepare a redeem when there is an associated monitored tx in the table.
	// The status of the tx is Timeout, so a new monitored tx should be added
	mt := RandMonitoredTx(Timeout, 1)
	mt.Id = redeem.RequestTxHash
	err = env.mgrdb.InsertMonitoredTx(mt)
	assert.NoError(t, err)
	ctx, cancel = context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()
	mts, err = env.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Len(t, mts, 2)
	for _, m := range mts {
		if m.Status == Timeout {
			assert.Equal(t, mt, m)
		} else {
			assert.Equal(t, redeem.RequestTxHash, m.Id)
		}
	}
}

func TestMonitorOnTimeout(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	blk, _ := env.sim.Chain.Backend.Client().BlockByNumber(context.Background(), nil)

	mt := RandMonitoredTx(Pending, 1)
	mt.SentAfter = blk.Hash()

	err := env.mgrdb.InsertMonitoredTx(mt)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()

	// generate [timeout + 1] blocks to trigger timeout
	for i := 0; i <= timeoutOnMonitoringPendingTxs; i++ {
		commit()
	}

	time.Sleep(frequencyToMonitorPendingTxs * 2)
	cancel()

	// status set as Timeout
	mts, err := env.mgrdb.GetMonitoredTxsById(mt.Id)
	assert.NoError(t, err)
	assert.Equal(t, Timeout, mts[0].Status)
}

// Main routine test procedures:
//  1. Start main routines of eth2btc state, eth tx manager, eth synchronizer, and mock wallet
//  2. Mock Mint twbtc tokens for account [1] and [2]
//     a. create two mint requests for [1] and [2]
//     b. insert the mint requests to the state db
//     c. mgr caputres "mint" automatically (every 0.5 second) and do real token mint on ethereum.
//  3. Two users approve twbtc tokens to be spent by bridge.
//  4. Two users Request redeems
//     [tx1]: from [1] with valid btc address
//     [tx2]: from [1] with invalid btc address
//     [tx3]: from [2] with valid btc address
//  5. Check for monitored Request tx -- Here we do not commit a new block for the sent txs
//     have row for [tx1, tx3]
//  6. Commit a new block
//  7. Check monitor pending Request txs
//     status == success for [tx1, tx3]
func TestMainRoutine(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file, common.MainNetParams())
	defer env.close()

	// shortcut
	commit := env.sim.Chain.Backend.Commit

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// 1. start main routines
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.st.Start(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.mgr.Start(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.sync.Sync(ctx)
	}()

	time.Sleep(1 * time.Second)

	// 2. mock mint twbtc tokens (directly insert Mint events into the state db)
	// TODO: in real life, btc side shall use observers to insert into statedb about "mint"
	// TODO: so btcTxId can be real.
	mints := []*state.Mint{
		{
			BtcTxId:  common.RandBytes32(),
			Receiver: env.sim.GetAuth(1).From.Bytes(),
			Amount:   big.NewInt(100),
		},
		{
			BtcTxId:  common.RandBytes32(),
			Receiver: env.sim.GetAuth(2).From.Bytes(),
			Amount:   big.NewInt(200),
		},
	}
	for _, mint := range mints {
		err := env.statedb.InsertMint(mint)
		assert.NoError(t, err)
	}
	// it only takes 0.5 for mgr to capture the un-minted,
	// and creates a token mint Tx on Ethereum automatcially.
	// so 1 second is long enough.
	time.Sleep(1 * time.Second)

	// Move ethereum blockchain forward to contain the token mint Tx.
	commit()
	// At this step, user's token balance on eth-side is credited.

	// 3. Two users (owner) approve twbtc tokens on ethereum (spender=bridge)
	env.sim.Approve(1, 90)  // 90 < 100
	env.sim.Approve(2, 100) // 100 < 200
	commit()
	printCurrBlockNumber(env, "approved")

	// 4. request the redeem on ethereum
	// from eth account at idx [1] with 60 satoshi to btc address at idx [0] (valid btc address)
	tx1, _ := env.sim.Request(env.sim.GetAuth(1), 1, 60, 0)
	// from eth account at idx [1] with 30 satoshi to btc adddress at idx [-1] (invalid btc address)
	tx2, _ := env.sim.Request(env.sim.GetAuth(1), 1, 30, -1)
	// from eth account at idx [2] with 100 satoshi to btc address at idx [1] (valid btc address)
	tx3, _ := env.sim.Request(env.sim.GetAuth(2), 2, 100, 1)
	commit()
	printCurrBlockNumber(env, "requested")

	// give time to process requested redeem
	time.Sleep(1 * time.Second)

	// 5. check for monitored tx
	mts, err := env.mgrdb.GetMonitoredTxsById(tx1)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx2)
	assert.NoError(t, err)
	assert.Len(t, mts, 0)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx3)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)

	// 6. commit a new block to allow the txs to be mined
	commit()
	printCurrBlockNumber(env, "prepared")
	time.Sleep(1 * time.Second)

	// 7. check monitor pending txs
	mts, err = env.mgrdb.GetMonitoredTxsById(tx1)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	assert.Equal(t, Success, mts[0].Status)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx3)
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	assert.Equal(t, Success, mts[0].Status)

	cancel()  // guess: cancel() ends sub go routines politely.
	wg.Wait() // wait for all the routines to complete.
}

func printCurrBlockNumber(env *testEnv, action string) {
	blk, _ := env.sim.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	logger.WithField("block", blk.Number()).Debug(action)
}
