package ethtxmanager

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"sync"
	"testing"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

const (
	// eth synchronizer config
	frequencyToCheckEthFinalizedBlock = 100 * time.Millisecond
	frequencyToCheckBtcFinalizedBlock = 100 * time.Millisecond

	// eth tx manager config
	frequencyToPrepareRedeem      = 500 * time.Millisecond
	frequencyToMint               = 500 * time.Millisecond
	frequencyToMonitorPendingTxs  = 500 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 10

	// blockInterval = 100 * time.Millisecond
)

type testEnv struct {
	sim *etherman.SimEtherman

	sqldb   *sql.DB
	statedb *state.StateDB
	st      *state.State
	mgrdb   *EthTxManagerDB
	mgr     *EthTxManager
	sync    *ethsync.Synchronizer
}

func newTestEnv(t *testing.T, file string) *testEnv {
	sim, err := etherman.NewSimEtherman()
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
	st, err := state.New(statedb, &state.Config{ChannelSize: 1})
	assert.NoError(t, err)

	// create a eth tx manager db
	mgrdb, err := NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)

	// create a eth synchronizer
	sync, err := ethsync.New(
		sim.Etherman,
		st,
		&ethsync.Config{
			FrequencyToCheckEthFinalizedBlock: frequencyToCheckEthFinalizedBlock,
			FrequencyToCheckBtcFinalizedBlock: frequencyToCheckBtcFinalizedBlock,
			BtcChainConfig:                    common.MainNetParams(),
			EthChainID:                        chainID,
		},
	)
	assert.NoError(t, err)

	// create a eth tx manager
	cfg := &Config{
		FrequencyToPrepareRedeem:      frequencyToPrepareRedeem,
		FrequencyToMint:               frequencyToMint,
		FrequencyToMonitorPendingTxs:  frequencyToMonitorPendingTxs,
		TimeoutOnWaitingForSignature:  timeoutOnWaitingForSignature,
		TimeoutOnWaitingForOutpoints:  timtoutOnWaitingForOutpoints,
		TimeoutOnMonitoringPendingTxs: timeoutOnMonitoringPendingTxs,
	}
	btcWallet := &MockBtcWallet{}
	schnorrWallet := &MockSchnorrThresholdWallet{sim}
	mgr, err := New(cfg, sim.Etherman, statedb, mgrdb, schnorrWallet, btcWallet)
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

func TestOnIsMinted(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file)
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	// mint the btcTxId on chain
	_, params := env.sim.Mint(1, 100)
	commit()

	mint := &state.Mint{
		BtcTxId:  params.BtcTxId,
		Receiver: params.Receiver,
		Amount:   common.BigIntClone(params.Amount),
	}
	err := env.statedb.InsertMint(mint)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToMint * 2)
	cancel()
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

	env := newTestEnv(t, file)
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

	env := newTestEnv(t, file)
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

	env := newTestEnv(t, file)
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

	env := newTestEnv(t, file)
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
//  2. Mint twbtc tokens for account [1] and [2]
//     a. create two mint requests for [1] and [2]
//     b. insert the mint requests to the state db
//  3. Approve twbtc tokens for the two users
//  4. Request redeem
//     [tx1]: from [1] with valid btc address
//     [tx2]: from [1] with invalid btc address
//     [tx3]: from [2] with valid btc address
//  5. Check for monitored tx -- Here we do not commit a new block for the sent txs
//     have row for [tx1, tx3]
//  6. commit a new block
//  7. Check monitor pending txs
//     status == success for [tx1, tx3]
func TestMainRoutine(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file)
	defer env.close()
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

	// 2. mint twbtc tokens
	mints := []*state.Mint{
		{
			BtcTxId:  common.RandBytes32(),
			Receiver: env.sim.GetAuth(1).From,
			Amount:   big.NewInt(100),
		},
		{
			BtcTxId:  common.RandBytes32(),
			Receiver: env.sim.GetAuth(2).From,
			Amount:   big.NewInt(200),
		},
	}
	for _, mint := range mints {
		err := env.statedb.InsertMint(mint)
		assert.NoError(t, err)
	}
	time.Sleep(1 * time.Second)
	commit()

	// 3. approve twbtc tokens
	env.sim.Approve(1, 90)
	env.sim.Approve(2, 100)
	commit()
	printCurrBlockNumber(env, "approved")

	// 4. request redeem
	tx1, _ := env.sim.Request(env.sim.GetAuth(1), 1, 60, 0)  // valid btc address
	tx2, _ := env.sim.Request(env.sim.GetAuth(1), 1, 30, -1) // invalid btc address
	tx3, _ := env.sim.Request(env.sim.GetAuth(2), 2, 100, 1) // valid btc address
	commit()
	printCurrBlockNumber(env, "requested")

	// give time to process requested redeem
	time.Sleep(1 * time.Second)

	// 5. check for monitored tx
	mts, err := env.mgrdb.GetMonitoredTxsById(tx1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mts, 1)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx2)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mts, 0)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mts, 1)

	// 6. commit a new block to allow the txs to be mined
	commit()
	printCurrBlockNumber(env, "prepared")
	time.Sleep(1 * time.Second)

	// 7. check monitor pending txs
	mts, err = env.mgrdb.GetMonitoredTxsById(tx1)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mts, 1)
	assert.Equal(t, Success, mts[0].Status)
	mts, err = env.mgrdb.GetMonitoredTxsById(tx3)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mts, 1)
	assert.Equal(t, Success, mts[0].Status)

	cancel()
	wg.Wait()
}

func printCurrBlockNumber(env *testEnv, txt string) {
	blk, _ := env.sim.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	logger.Debugf("%s at block=%v", txt, blk.Number())
}
