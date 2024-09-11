package synchronizer

import (
	"context"
	"math/big"
	"testing"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
)

func TestSync(t *testing.T) {
	env := etherman.NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}

	mockE2BState := NewMockEth2BtcState()
	mockB2EState := NewMockBtc2EthState()

	cfg := &EthSyncConfig{
		Etherman:                     env.Etherman,
		CheckFinalizedTickerInterval: 100 * time.Millisecond,
		Btc2EthState:                 mockB2EState,
		Eth2BtcState:                 mockE2BState,
		LastFinalizedBLock:           big.NewInt(0),
	}

	sync := NewEthSynchronizer(cfg)
	if sync == nil {
		t.Fatal("failed to create synchronizer")
	}

	common.Debug = true
	logger.Debug("DEBUG ON")
	defer func() {
		logger.Debug("DEBUG OFF")
		common.Debug = false
	}()

	ctx, cancel := context.WithCancel(context.Background())
	sync.Sync(ctx)

	cancel()
}

// func sendTxs(t *testing.T, env *etherman.TestEnv) {
// 	sim := env.Sim
// 	etherman := env.Etherman

// 	mintParams := env.GenMintParams(&etherman.ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
// 	err := etherman.Mint(mintParams)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	prepareParams := env.GenPrepareParams(&etherman.ParamConfig{sender: 3, requester: 4, amount: big.NewInt(400)})
// 	err = etherman.RedeemPrepare(prepareParams)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	sim.Backend.Commit()

// 	err = etherman.TWBTCApprove(sim.Accounts[1], big.NewInt(80))
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	requestParams := env.GenRequestParams(&etherman.ParamConfig{Sender: 1, Amount: big.NewInt(80)})
// 	if requestParams == nil {
// 		t.Fatal("failed to generate request params")
// 	}
// 	err = etherman.RedeemRequest(requestParams)
// 	assert.NoError(t, err)
// 	sim.Backend.Commit()
// }
