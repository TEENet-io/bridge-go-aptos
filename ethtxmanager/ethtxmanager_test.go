package ethtxmanager

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

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
	frequencyToMonitorPendingTxs  = 800 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 1
)

type testEnv struct {
	ctx    context.Context
	cancel context.CancelFunc

	sim *etherman.SimEtherman

	sqldb   *sql.DB
	statedb *state.StateDB
	st      *state.State
	mgrdb   *EthTxManagerDB
	mgr     *EthTxManager
	sync    *ethsync.Synchronizer
}

func newTestEnv(t *testing.T) *testEnv {
	ctx, cancel := context.WithCancel(context.Background())
	sim, err := etherman.NewSimEtherman()
	assert.NoError(t, err)
	chainID, err := sim.Etherman.Client().ChainID(ctx)
	assert.NoError(t, err)

	// create a sql db
	sqldb, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	// create a eth2btc state db
	statedb, err := state.NewStateDB(sqldb)
	assert.NoError(t, err)
	_, _, err = statedb.GetRedeem(ethcommon.Hash{}, state.RedeemStatusCompleted)
	assert.NoError(t, err)

	// create a eth2btc state from the eth2btc statedb
	st, err := state.New(statedb, &state.Config{ChannelSize: 1})
	assert.NoError(t, err)

	// create a eth tx manager db
	mgrdb, err := NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)
	_, _, err = mgrdb.GetSignatureRequestByRequestTxHash(ethcommon.Hash{})
	assert.NoError(t, err)
	_, _, err = mgrdb.GetMonitoredTxByRequestTxHash(ethcommon.Hash{})
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
	mgr, err := New(ctx, cfg, sim.Etherman, statedb, mgrdb, schnorrWallet, btcWallet)
	assert.NoError(t, err)

	return &testEnv{ctx, cancel, sim, sqldb, statedb, st, mgrdb, mgr, sync}
}

func (env *testEnv) close() {
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
//  8. Check monitor pending txs
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
		env.st.Start(env.ctx)
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
	fmt.Println("minting twbtc tokens")
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

	// 3. approve twbtc tokens
	fmt.Println("approving twbtc tokens")
	env.sim.Approve(1, 90)
	env.sim.Approve(2, 100)
	fmt.Println("committing")
	commit()

	// 4. request redeem
	fmt.Println("requesting redeem")
	tx1, _ := env.sim.Request(env.sim.GetAuth(1), 1, 60, 0)  // valid btc address
	tx2, _ := env.sim.Request(env.sim.GetAuth(1), 1, 30, -1) // invalid btc address
	tx3, _ := env.sim.Request(env.sim.GetAuth(2), 2, 100, 1) // valid btc address
	fmt.Println("committing")
	commit()

	// give time to process
	fmt.Println("wait for 5 seconds")
	time.Sleep(5 * time.Second)

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
	fmt.Println("committing")
	commit()

	fmt.Println("wait for 5 seconds")
	time.Sleep(5 * time.Second)

	// 8. check monitor pending txs
	mtxs, err := env.mgrdb.GetAllMonitoredTx()
	assert.NoError(t, err)
	assert.Len(t, mtxs, 0)

	env.cancel()
	wg.Wait()
}
