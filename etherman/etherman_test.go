package etherman

import (
	"context"
	"math/big"
	"testing"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"
)

func TestGetEventLogs(t *testing.T) {
	env := NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}
	sim := env.Sim
	etherman := env.Etherman

	mintParams := env.GenMintParams(&ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
	if mintParams == nil {
		t.Fatal("failed to generate mint params")
	}
	err := etherman.Mint(mintParams)
	assert.NoError(t, err)

	prepareParams := env.GenPrepareParams(&ParamConfig{Sender: 3, Requester: 4, Amount: big.NewInt(400)})
	if prepareParams == nil {
		t.Fatal("failed to generate prepare params")
	}
	err = etherman.RedeemPrepare(prepareParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	num := curentBlockNum(t, env)

	minted, requested, prepared, err := etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 1)
	assert.Len(t, requested, 0)
	assert.Len(t, prepared, 1)

	checkMintedEvent(t, &minted[0], mintParams)
	checkPreparedEvent(t, &prepared[0], prepareParams)

	err = etherman.TWBTCApprove(sim.Accounts[1], big.NewInt(80))
	assert.NoError(t, err)
	sim.Backend.Commit()

	requestParams := env.GenRequestParams(&ParamConfig{Sender: 1, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	err = etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	num = curentBlockNum(t, env)
	minted, requested, prepared, err = etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 0)
	assert.Len(t, requested, 1)
	assert.Len(t, prepared, 0)

	checkRequestedEvent(t, &requested[0], requestParams)
}

func TestRedeemPrepare(t *testing.T) {
	env := NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}
	sim := env.Sim
	etherman := env.Etherman

	params := env.GenPrepareParams(&ParamConfig{Sender: 0, Requester: 1, Amount: big.NewInt(100)})
	if params == nil {
		t.Fatal("failed to generate prepare params")
	}
	err := etherman.RedeemPrepare(params)
	assert.NoError(t, err)
	sim.Backend.Commit()
}

func TestRedeemRequest(t *testing.T) {
	env := NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}
	sim := env.Sim
	etherman := env.Etherman

	// Mint tokens
	minParams := env.GenMintParams(&ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
	if minParams == nil {
		t.Fatal("failed to generate mint params")
	}
	err := etherman.Mint(minParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	// Approve tokens to bridge
	err = etherman.TWBTCApprove(sim.Accounts[1], big.NewInt(80))
	assert.NoError(t, err)
	sim.Backend.Commit()

	allowance, err := etherman.TWBTCAllowance(sim.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(80), allowance)

	// Request redeem
	requestParams := env.GenRequestParams(&ParamConfig{Sender: 1, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	err = etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	balance, err := etherman.TWBTCBalanceOf(sim.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(20), balance)
}

func TestMint(t *testing.T) {
	env := NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}
	sim := env.Sim
	etherman := env.Etherman

	params := env.GenMintParams(&ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
	if params == nil {
		t.Fatal("failed to generate mint params")
	}
	err := etherman.Mint(params)
	assert.NoError(t, err)
	sim.Backend.Commit()

	balance, err := etherman.TWBTCBalanceOf(params.Receiver)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), balance)
}

func TestGetLatestFinalizedBlockNumber(t *testing.T) {
	etherman, err := NewEtherman(&Config{
		URL:                   "https://mainnet.infura.io/v3/f37af697a9dd4cbfa7e22aaacce33e50",
		BridgeContractAddress: "0x0000000000000000000000000000000000000000",
	})
	assert.NoError(t, err)

	b, err := etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotZero(t, b)
}

func TestDebugGetLatestFinalizedBlockNumber(t *testing.T) {
	env := NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}
	etherman := env.Etherman

	common.Debug = true
	logger.Debug("DEBUG ON")
	defer func() {
		common.Debug = false
		logger.Debug("DEBUG OFF")
	}()

	b, err := etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, b, big.NewInt(1))

	env.Sim.Backend.Commit()

	b, err = etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, b, big.NewInt(2))
}

func curentBlockNum(t *testing.T, env *TestEnv) *big.Int {
	block, err := env.Sim.Backend.Client().BlockNumber(context.Background())
	assert.NoError(t, err)
	return big.NewInt(int64(block))
}

func checkMintedEvent(t *testing.T, ev *MintedEvent, params *MintParams) {
	assert.Equal(t, ev.BtcTxId, params.BtcTxId)
	assert.Equal(t, ev.Receiver.String(), params.Receiver.String())
	assert.Equal(t, ev.Amount, params.Amount)
}

func checkPreparedEvent(t *testing.T, ev *RedeemPreparedEvent, params *PrepareParams) {
	assert.Equal(t, ev.EthTxHash, params.TxHash)
	assert.Equal(t, ev.Requester.String(), params.Requester.String())
	assert.Equal(t, ev.Amount, params.Amount)
	for i, txId := range ev.OutpointTxIds {
		assert.Equal(t, txId, params.OutpointTxIds[i])
	}
	for i, idx := range ev.OutpointIdxs {
		assert.Equal(t, idx, params.OutpointIdxs[i])
	}
}

func checkRequestedEvent(t *testing.T, ev *RedeemRequestedEvent, params *RequestParams) {
	assert.Equal(t, ev.Sender.String(), params.Auth.From.String())
	assert.Equal(t, ev.Amount, params.Amount)
	assert.Equal(t, ev.Receiver, string(params.Receiver))
}
