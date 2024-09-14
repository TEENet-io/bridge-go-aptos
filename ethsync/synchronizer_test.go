package ethsync

import (
	"context"
	"math/big"
	"testing"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/stretchr/testify/assert"
	"github.com/vechain/thor/co"
)

func TestSync(t *testing.T) {
	env := etherman.NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}

	mockE2BState := NewMockEth2BtcState()
	mockB2EState := NewMockBtc2EthState()

	cfg := &Config{
		Etherman:                          env.Etherman,
		CheckLatestFinalizedBlockInterval: 100 * time.Millisecond,
		Btc2EthState:                      mockB2EState,
		Eth2BtcState:                      mockE2BState,
	}

	synchronizer := New(cfg)
	if synchronizer == nil {
		t.Fatal("failed to create synchronizer")
	}

	common.Debug = true
	logger.Debug("DEBUG ON")
	defer func() {
		logger.Debug("DEBUG OFF")
		common.Debug = false
	}()

	ctx, cancel := context.WithCancel(context.Background())

	var goes co.Goes
	goes.Go(func() { go mockB2EState.Start(ctx) })
	goes.Go(func() { go mockE2BState.Start(ctx) })
	goes.Go(func() { go synchronizer.Sync(ctx) })

	sendTxs(t, env)

	time.Sleep(200 * time.Millisecond)

	cancel()

	goes.Wait()

	blk, err := env.Sim.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.NoError(t, err)

	assert.Equal(t, blk.Number(), mockE2BState.lastFinalized)
	assert.Equal(t, 1, len(mockE2BState.requestedEv))
	assert.Equal(t, 1, len(mockE2BState.preparedEv))
	assert.Equal(t, 1, len(mockB2EState.mintedEv))
}

func sendTxs(t *testing.T, env *etherman.TestEnv) {
	mintParams := env.GenMintParams(&etherman.ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
	_, err := env.Etherman.Mint(mintParams)
	if err != nil {
		t.Fatal(err)
	}

	prepareParams := env.GenPrepareParams(&etherman.ParamConfig{Sender: 3, Requester: 4, Amount: big.NewInt(400)})
	_, err = env.Etherman.RedeemPrepare(prepareParams)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	err = env.Etherman.TWBTCApprove(env.Sim.Accounts[1], big.NewInt(80))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	requestParams := env.GenRequestParams(&etherman.ParamConfig{Sender: 1, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	_, err = env.Etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()
}
