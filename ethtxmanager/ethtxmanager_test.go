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
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

const (
	frequencyToGetUnpreparedRedeem = 500 * time.Millisecond
	timeoutOnWaitingForSignature   = 1 * time.Second
	frequencyToCheckFinalizedBlock = 100 * time.Millisecond
	blockInterval                  = 100 * time.Millisecond
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

	e2bst, err := eth2btcstate.New(statedb, &eth2btcstate.Config{ChannelSize: 1})
	assert.NoError(t, err)
	b2est := ethsync.NewMockBtc2EthState()

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

	mgrdb, err := NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)

	cfg := &Config{
		FrequencyToGetUnpreparedRedeem: frequencyToGetUnpreparedRedeem,
		TimeoutOnWaitingForSignature:   timeoutOnWaitingForSignature,
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

func TestPrepareRedeem(t *testing.T) {
	common.Debug = true
	defer func() {
		common.Debug = false
	}()

	env := newTestEnv(t)
	defer env.close()

	wg := &sync.WaitGroup{}

	// Start
	// 		eth2btc state for storing redeem info;
	// 		eth tx manager for preparing redeem;
	// 		eth synchronizer for monitoring bridge events; and
	// 		mock wallet for generating schnorr signature
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := env.e2bst.Start(env.ctx)
		panic(err)
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
		growChain(env)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		env.sync.Sync(env.ctx)
	}()

	// mint twbtc tokens
	env.sim.Mint(1, 100)
	env.sim.Mint(2, 200)

	// request redeem
	tx1, _ := env.sim.Request(1, 50, 0)  // valid btc address
	tx2, _ := env.sim.Request(1, 50, -1) // invalid btc address
	tx3, _ := env.sim.Request(2, 100, 1) // valid btc address

	// Check for signature request
	sr1, err := env.mgrdb.GetSignatureRequestByRequestTxHash(tx1)
	assert.NoError(t, err)
	assert.NotNil(t, sr1)
	sr2, err := env.mgrdb.GetSignatureRequestByRequestTxHash(tx2)
	assert.NoError(t, err)
	assert.Nil(t, sr2)
	sr3, err := env.mgrdb.GetSignatureRequestByRequestTxHash(tx3)
	assert.NoError(t, err)
	assert.NotNil(t, sr3)

	time.Sleep(5 * time.Second)
	env.cancel()
	wg.Wait()
}

func growChain(env *testEnv) {
	for {
		select {
		case <-env.ctx.Done():
			return
		case <-time.After(blockInterval):
			env.sim.Chain.Backend.Commit()
		}
	}
}
