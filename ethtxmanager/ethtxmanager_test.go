package ethtxmanager

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

const (
	frequencyToGetUnpreparedRedeem = 500 * time.Millisecond
	frequencyToCheckFinalizedBlock = 500 * time.Millisecond
	frequencyToMonitorPendingTxs   = 500 * time.Millisecond
	timeoutOnWaitingForSignature   = 1 * time.Second
	timtoutOnWaitingForOutpoints   = 1 * time.Second
	blockInterval                  = 10 * time.Second
)

type testEnv struct {
	ctx    context.Context
	cancel context.CancelFunc

	sim *etherman.SimEtherman

	sqldb   *sql.DB
	statedb *eth2btcstate.StateDB
	e2bst   *eth2btcstate.State
	b2est   *ethsync.MockBtc2EthState
	mgrdb   *EthTxManagerDB
	mgr     *EthTxManager
	ch      chan *SignatureRequest
	sync    *ethsync.Synchronizer
}

func newTestEnv(t *testing.T) *testEnv {
	ctx, cancel := context.WithCancel(context.Background())
	sim, err := etherman.NewSimEtherman()
	assert.NoError(t, err)
	chainID, err := sim.Etherman.Client().ChainID(ctx)
	assert.NoError(t, err)

	sqldb, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	statedb, err := eth2btcstate.NewStateDB(sqldb)
	assert.NoError(t, err)

	// test statedb
	_, _, err = statedb.Get(ethcommon.Hash{}, eth2btcstate.RedeemStatusCompleted)
	assert.NoError(t, err)

	e2bst, err := eth2btcstate.New(statedb, &eth2btcstate.Config{ChannelSize: 1})
	assert.NoError(t, err)
	b2est := ethsync.NewMockBtc2EthState()

	mgrdb, err := NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)
	// test mgrdb
	_, _, err = mgrdb.GetSignatureRequestByRequestTxHash(ethcommon.Hash{})
	assert.NoError(t, err)
	_, _, err = mgrdb.GetMonitoredTxByRequestTxHash(ethcommon.Hash{})
	assert.NoError(t, err)

	sync, err := ethsync.New(
		sim.Etherman,
		e2bst,
		b2est,
		&ethsync.Config{
			FrequencyToCheckFinalizedBlock: frequencyToCheckFinalizedBlock,
			BtcChainConfig:                 common.MainNetParams(),
			EthChainID:                     chainID,
		},
	)
	assert.NoError(t, err)

	cfg := &Config{
		FrequencyToGetUnpreparedRedeem: frequencyToGetUnpreparedRedeem,
		FrequencyToMonitorPendingTxs:   frequencyToMonitorPendingTxs,
		TimeoutOnWaitingForSignature:   timeoutOnWaitingForSignature,
		TimeoutOnWaitingForOutpoints:   timtoutOnWaitingForOutpoints,
	}

	ch := make(chan *SignatureRequest, 1)

	btcWallet := &MockBtcWallet{}
	schnorrWallet := &MockSchnorrThresholdWallet{sim}

	mgr, err := New(ctx, cfg, sim.Etherman, statedb, mgrdb, schnorrWallet, btcWallet)
	assert.NoError(t, err)

	return &testEnv{ctx, cancel, sim, sqldb, statedb, e2bst, b2est, mgrdb, mgr, ch, sync}
}

func (env *testEnv) close() {
	env.cancel()
	env.sqldb.Close()
	env.mgrdb.Close()
	env.statedb.Close()
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
//  8. Monitor pending txs
//     no rows after observing the txs are mined
func TestMainRoutine(t *testing.T) {
	common.Debug = true
	defer func() {
		common.Debug = false
	}()

	env := newTestEnv(t)
	defer env.close()
	commit := env.sim.Chain.Backend.Commit

	wg := &sync.WaitGroup{}

	// 1. start main routines
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.e2bst.Start(env.ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.b2est.Start(env.ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.mgr.Start(env.ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.sync.Sync(env.ctx)
	}()

	// 2. mint twbtc tokens
	env.sim.Mint(1, 100)
	env.sim.Mint(2, 200)
	commit()

	// 3. approve twbtc tokens
	env.sim.Approve(1, 90)
	env.sim.Approve(2, 100)
	commit()

	// 4. request redeem
	tx1, _ := env.sim.Request(env.sim.GetAuth(1), 1, 60, 0)  // valid btc address
	tx2, _ := env.sim.Request(env.sim.GetAuth(1), 1, 30, -1) // invalid btc address
	tx3, _ := env.sim.Request(env.sim.GetAuth(2), 2, 100, 1) // valid btc address
	commit()

	// give time to process
	time.Sleep(10 * time.Second)

	// 5. check for signature request
	// tx1
	sr1, ok, err := env.mgrdb.GetSignatureRequestByRequestTxHash(tx1)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, common.Verify(env.sim.Sk.PubKey().X().Bytes(), sr1.SigningHash[:], sr1.Rx, sr1.S))
	// tx2
	_, ok, err = env.mgrdb.GetSignatureRequestByRequestTxHash(tx2)
	assert.NoError(t, err)
	assert.False(t, ok)
	// tx3
	_, ok, err = env.mgrdb.GetSignatureRequestByRequestTxHash(tx3)
	assert.NoError(t, err)
	assert.True(t, ok)

	// 6. check for monitored tx
	// tx1
	_, ok, err = env.mgrdb.GetMonitoredTxByRequestTxHash(tx1)
	assert.NoError(t, err)
	assert.True(t, ok)
	// tx2
	_, ok, err = env.mgrdb.GetMonitoredTxByRequestTxHash(tx2)
	assert.NoError(t, err)
	assert.False(t, ok)
	// tx3
	_, ok, err = env.mgrdb.GetMonitoredTxByRequestTxHash(tx3)
	assert.NoError(t, err)
	assert.True(t, ok)

	// 7. commit a new block to allow the txs to be mined
	commit()

	time.Sleep(10 * time.Second)

	// 8. monitor pending txs
	mtxs, err := env.mgrdb.GetAllMonitoredTx()
	assert.NoError(t, err)
	assert.Len(t, mtxs, 0)

	env.cancel()
	wg.Wait()
}

// func growChain(env *testEnv) {
// 	for {
// 		select {
// 		case <-env.ctx.Done():
// 			return
// 		case <-time.After(blockInterval):
// 			env.sim.Chain.Backend.Commit()
// 		}
// 	}
// }
