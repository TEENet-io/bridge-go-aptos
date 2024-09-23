package etherman

import (
	"context"
	"math/big"
	"testing"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestNonce(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)

	env.Mint(1, 100)
	env.Chain.Backend.Commit()
	nonce, err := env.Etherman.ethClient.PendingNonceAt(context.Background(), env.Chain.Accounts[0].From)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), nonce)
	env.Mint(2, 100)
	nonce, err = env.Etherman.ethClient.PendingNonceAt(context.Background(), env.Chain.Accounts[0].From)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), nonce)
}

func TestIsPrepared(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
	etherman := env.Etherman
	commit := env.Chain.Backend.Commit

	params := env.GenPrepareParams(&ParamConfig{Requester: 1, Amount: big.NewInt(100), OutpointNum: 2})
	assert.NotNil(t, params)

	_, err = etherman.RedeemPrepare(params)
	assert.NoError(t, err)
	commit()

	prepared, err := etherman.IsPrepared(params.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, prepared)
}

func TestIsMinted(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
	etherman := env.Etherman
	commit := env.Chain.Backend.Commit

	params := env.GenMintParams(&ParamConfig{Receiver: 1, Amount: big.NewInt(100)})
	assert.NotNil(t, params)
	_, err = etherman.Mint(params)
	assert.NoError(t, err)
	commit()

	minted, err := etherman.IsMinted(params.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, minted)
}

func TestGetEventLogs(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)

	commit := env.Chain.Backend.Commit

	_, mintParams := env.Mint(1, 100)
	txHash, prepareParams := env.Prepare(4, 400, 0, 1)
	commit()

	num := curentBlockNum(t, env)

	minted, requested, prepared, err := env.Etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 1)
	assert.Len(t, requested, 0)
	assert.Len(t, prepared, 1)

	assert.Equal(t, txHash, ethcommon.BytesToHash(prepared[0].TxHash[:]))
	checkMintedEvent(t, &minted[0], mintParams)
	checkPreparedEvent(t, &prepared[0], prepareParams)

	requester := 1

	env.Approve(1, 80)
	commit()

	txHash, requestParams := env.Request(env.GetAuth(requester), 1, 80, 0)
	commit()

	num = curentBlockNum(t, env)
	minted, requested, prepared, err = env.Etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 0)
	assert.Len(t, requested, 1)
	assert.Len(t, prepared, 0)

	assert.Equal(t, txHash, ethcommon.BytesToHash(requested[0].TxHash[:]))
	checkRequestedEvent(t, &requested[0], requestParams)
}

func TestRedeemPrepare(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
	etherman := env.Etherman
	commit := env.Chain.Backend.Commit

	params := env.GenPrepareParams(&ParamConfig{Requester: 1, Amount: big.NewInt(100), OutpointNum: 3})
	assert.NotNil(t, params)
	_, err = etherman.RedeemPrepare(params)
	assert.NoError(t, err)
	commit()
}

func TestRedeemRequest(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
	etherman := env.Etherman
	commit := env.Chain.Backend.Commit

	// Mint tokens
	env.Mint(1, 100)
	commit()

	// Approve tokens to bridge
	env.Approve(1, 80)
	commit()

	allowance, err := etherman.TWBTCAllowance(env.Chain.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(80), allowance)

	// Request redeem
	requester := 1
	requestParams := env.GenRequestParams(&ParamConfig{Requester: requester, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	_, err = etherman.RedeemRequest(env.GetAuth(requester), requestParams)
	assert.NoError(t, err)
	commit()

	balance, err := etherman.TWBTCBalanceOf(env.Chain.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(20), balance)
}

func TestMint(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
	etherman := env.Etherman
	commit := env.Chain.Backend.Commit

	params := env.GenMintParams(&ParamConfig{Receiver: 1, Amount: big.NewInt(100)})
	assert.NotNil(t, params)
	_, err = etherman.Mint(params)
	assert.NoError(t, err)
	commit()

	balance, err := etherman.TWBTCBalanceOf(params.Receiver)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), balance)
}

func TestGetLatestFinalizedBlockNumber(t *testing.T) {
	URL := "https://mainnet.infura.io/v3/f37af697a9dd4cbfa7e22aaacce33e50"
	client, err := ethclient.Dial(URL)
	assert.NoError(t, err)
	etherman := &Etherman{ethClient: client}

	b, err := etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotZero(t, b)
}

func TestDebugGetLatestFinalizedBlockNumber(t *testing.T) {
	env, err := NewSimEtherman()
	assert.NoError(t, err)
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

	env.Chain.Backend.Commit()

	b, err = etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, b, big.NewInt(2))
}

func curentBlockNum(t *testing.T, env *SimEtherman) *big.Int {
	block, err := env.Chain.Backend.Client().BlockNumber(context.Background())
	assert.NoError(t, err)
	return big.NewInt(int64(block))
}

func checkMintedEvent(t *testing.T, ev *MintedEvent, params *MintParams) {
	assert.Equal(t, ev.BtcTxId[:], params.BtcTxId.Bytes())
	assert.Equal(t, ev.Receiver, params.Receiver)
	assert.Equal(t, ev.Amount, params.Amount)
}

func checkPreparedEvent(t *testing.T, ev *RedeemPreparedEvent, params *PrepareParams) {
	assert.Equal(t, ev.EthTxHash[:], params.RequestTxHash.Bytes())
	assert.Equal(t, ev.Requester.String(), params.Requester.String())
	assert.Equal(t, ev.Amount, params.Amount)
	for i, txId := range ev.OutpointTxIds {
		assert.Equal(t, txId[:], params.OutpointTxIds[i].Bytes())
	}
	for i, idx := range ev.OutpointIdxs {
		assert.Equal(t, idx, params.OutpointIdxs[i])
	}
}

func checkRequestedEvent(t *testing.T, ev *RedeemRequestedEvent, params *RequestParams) {
	assert.Equal(t, ev.Amount, params.Amount)
	assert.Equal(t, ev.Receiver, params.Receiver)
}
