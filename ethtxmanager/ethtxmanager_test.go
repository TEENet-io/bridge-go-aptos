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
	frequencyToMint               = 200 * time.Millisecond
	frequencyToMonitorPendingTxs  = 500 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 10

	blockInterval = 100 * time.Millisecond
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
	mtxs, err := env.mgrdb.GetMonitoredTxs()
	assert.NoError(t, err)
	assert.Len(t, mtxs, 0)
}

func TestOnExistingMonitoringTx(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file)
	defer env.close()

	// insert a requested redeem to trigger preparing prodecure
	redeem := &state.Redeem{
		RequestTxHash: common.RandBytes32(),
		Requester:     common.RandEthAddress(),
		Receiver:      "btc_adress",
		Amount:        big.NewInt(100),
		Status:        state.RedeemStatusRequested,
	}
	err := env.statedb.InsertAfterRequested(redeem)
	assert.NoError(t, err)

	// insert a monitoring tx with its id = redeem.RequestTxHash
	mt := &monitoredTx{
		TxHash:    common.RandBytes32(),
		Id:        redeem.RequestTxHash,
		SentAfter: common.RandBytes32(),
	}
	err = env.mgrdb.InsertMonitoredTx(mt)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() { err = env.mgr.Start(ctx) }()

	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()

	// no new tx created for monitoring
	mtxs, err := env.mgrdb.GetMonitoredTxs()
	assert.NoError(t, err)
	assert.Len(t, mtxs, 1)
	assert.Equal(t, mt, mtxs[0])
}

func TestOnExistingSignatureRequest(t *testing.T) {
	common.Debug = true
	file := randFile()
	defer func() {
		common.Debug = false
		os.Remove(file)
	}()

	env := newTestEnv(t, file)
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	// mint and approve on chain
	env.sim.Mint(1, 100)
	commit()
	env.sim.Approve(1, 100)
	commit()
	tx, reqParams := env.sim.Request(env.sim.GetAuth(1), 1, 100, 1)
	commit()

	// generate prepare params
	prepParams := &etherman.PrepareParams{
		RequestTxHash: tx,
		Requester:     env.sim.GetAuth(1).From,
		Receiver:      reqParams.Receiver,
		Amount:        common.BigIntClone(reqParams.Amount),
		OutpointTxIds: []ethcommon.Hash{common.RandBytes32()},
		OutpointIdxs:  []uint16{0},
	}

	// insert the requested redeem to the state db
	redeem := &state.Redeem{
		RequestTxHash: prepParams.RequestTxHash,
		Requester:     prepParams.Requester,
		Receiver:      prepParams.Receiver,
		Amount:        common.BigIntClone(prepParams.Amount),
		Status:        state.RedeemStatusRequested,
	}
	err := env.statedb.InsertAfterRequested(redeem)
	assert.NoError(t, err)

	// generate signature request
	sr := &SignatureRequest{
		RequestTxHash: prepParams.RequestTxHash,
		SigningHash:   prepParams.SigningHash(),
		Outpoints: []state.Outpoint{
			{
				TxId: prepParams.OutpointTxIds[0],
				Idx:  prepParams.OutpointIdxs[0],
			},
		},
	}
	rx, s, err := env.sim.Sign(sr.SigningHash[:])
	assert.NoError(t, err)
	sr.Rx = rx
	sr.S = s

	// insert the signature request to the db
	err = env.mgrdb.InsertSignatureRequest(sr)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToPrepareRedeem * 2)
	cancel()

	// new tx created for monitoring
	mtxs, err := env.mgrdb.GetMonitoredTxs()
	assert.NoError(t, err)
	assert.Len(t, mtxs, 1)
	assert.Equal(t, sr.RequestTxHash, mtxs[0].Id)
}

func TestMonintorOnMined(t *testing.T) {
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
	sentAfter := blk.Hash()

	tx, params := env.sim.Prepare(1, 100, 1, 1)
	commit()

	// insert a monitoring tx
	mt := &monitoredTx{
		TxHash:    tx,
		Id:        params.RequestTxHash,
		SentAfter: sentAfter,
	}
	err := env.mgrdb.InsertMonitoredTx(mt)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = env.mgr.Start(ctx) }()
	time.Sleep(frequencyToMonitorPendingTxs * 2)
	cancel()

	// monitored tx removed
	_, ok, err := env.mgrdb.GetMonitoredTxById(mt.Id)
	assert.NoError(t, err)
	assert.False(t, ok)
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

	mt := &monitoredTx{
		TxHash:    common.RandBytes32(),
		Id:        common.RandBytes32(),
		SentAfter: blk.Hash(),
	}
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

	// monitored tx removed
	_, ok, err := env.mgrdb.GetMonitoredTxById(mt.Id)
	assert.NoError(t, err)
	assert.False(t, ok)
}

// Main routine test procedures:
//  1. Start main routines of eth2btc state, eth tx manager, eth synchronizer, and mock wallet
//  2. Mint twbtc tokens for account [1] and [2]
//  3. Approve twbtc tokens for the two users
//  4. Request redeem
//     [tx1]: from [1] with valid btc address
//     [tx2]: from [1] with invalid btc address
//     [tx3]: from [2] with valid btc address
//  5. Check for signature request
//     have row for [tx1, tx3]
//     no row for [tx2]
//  6. Check for monitored tx -- Here we do not commit a new block for the sent txs
//     have row for [tx1, tx3]
//  7. commit a new block
//  8. Check monitor pending txs
//     no rows after observing the txs are mined
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
	_, params := env.sim.Mint(1, 100)
	env.statedb.InsertMint(&state.Mint{
		BtcTxID:  params.BtcTxId,
		Receiver: params.Receiver,
		Amount:   common.BigIntClone(params.Amount),
	})
	_, params = env.sim.Mint(2, 200)
	env.statedb.InsertMint(&state.Mint{
		BtcTxID:  params.BtcTxId,
		Receiver: params.Receiver,
		Amount:   common.BigIntClone(params.Amount),
	})
	commit()
	printCurrBlockNumber(env, "minted")

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

	// 5. check for saved signature request
	// tx1
	sr1, ok, err := env.mgrdb.GetSignatureRequestByRequestTxHash(tx1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok)
	assert.True(t, common.Verify(env.sim.Sk.PubKey().X().Bytes(), sr1.SigningHash[:], sr1.Rx, sr1.S))
	// tx2
	_, ok, err = env.mgrdb.GetSignatureRequestByRequestTxHash(tx2)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok)
	// tx3
	_, ok, err = env.mgrdb.GetSignatureRequestByRequestTxHash(tx3)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok)

	// 6. check for monitored tx
	// tx1
	_, ok, err = env.mgrdb.GetMonitoredTxById(tx1)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok)
	// tx2
	_, ok, err = env.mgrdb.GetMonitoredTxById(tx2)
	if err != nil {
		t.Fatal(err)
	}
	assert.False(t, ok)
	// tx3
	_, ok, err = env.mgrdb.GetMonitoredTxById(tx3)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, ok)

	// 7. commit a new block to allow the txs to be mined
	commit()
	printCurrBlockNumber(env, "prepared")
	time.Sleep(1 * time.Second)

	// 8. check monitor pending txs
	mtxs, err := env.mgrdb.GetMonitoredTxs()
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, mtxs, 0)

	cancel()
	wg.Wait()
}

func printCurrBlockNumber(env *testEnv, txt string) {
	blk, _ := env.sim.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	logger.Debugf("%s at block=%v", txt, blk.Number())
}
