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
)

func TestSync(t *testing.T) {
	env := etherman.NewTestEnv()
	if env == nil {
		t.Fatal("failed to create test environment")
	}

	chainID, err := env.Etherman.Client().ChainID(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, chainID, big.NewInt(1337))

	mockE2BState := NewMockEth2BtcState()
	mockB2EState := NewMockBtc2EthState()

	cfg := &Config{
		FrequencyToCheckFinalizedBlock: 100 * time.Millisecond,
		BtcChainConfig:                 common.MainNetParams(),
		EthChainID:                     chainID,
	}

	synchronizer, err := New(env.Etherman, mockE2BState, mockB2EState, cfg)
	assert.NoError(t, err)
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

	go mockB2EState.Start(ctx)
	go mockE2BState.Start(ctx)
	go synchronizer.Sync(ctx)

	mintedEvs, reqeustedEvs, preparedEvs := sendTxs(t, env)

	time.Sleep(200 * time.Millisecond)

	cancel()

	blk, err := env.Sim.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.NoError(t, err)

	assert.Equal(t, blk.Number(), mockE2BState.lastFinalized)
	assert.Equal(t, 2, len(mockE2BState.requestedEv))
	assert.Equal(t, 1, len(mockE2BState.preparedEv))
	assert.Equal(t, 1, len(mockB2EState.mintedEv))

	assert.Equal(t, mintedEvs[0], mockB2EState.mintedEv[0])
	assert.Equal(t, reqeustedEvs[0], mockE2BState.requestedEv[0])
	assert.Equal(t, reqeustedEvs[1], mockE2BState.requestedEv[1])
	assert.Equal(t, preparedEvs[0], mockE2BState.preparedEv[0])
}

func sendTxs(t *testing.T, env *etherman.TestEnv) (
	mintedEvs []*MintedEvent,
	requestedEvs []*RedeemRequestedEvent,
	preparedEvs []*RedeemPreparedEvent,
) {
	mintParams := env.GenMintParams(&etherman.ParamConfig{Deployer: 0, Receiver: 1, Amount: big.NewInt(100)})
	tx, err := env.Etherman.Mint(mintParams)
	if err != nil {
		t.Fatal(err)
	}

	mintedEvs = append(mintedEvs, &MintedEvent{
		MintedTxHash: [32]byte(tx.Hash().Bytes()),
		BtcTxId:      mintParams.BtcTxId,
		Amount:       new(big.Int).Set(mintParams.Amount),
		Receiver:     mintParams.Receiver,
	})

	prepareParams := env.GenPrepareParams(&etherman.ParamConfig{Sender: 3, Requester: 4, Amount: big.NewInt(400)})
	tx, err = env.Etherman.RedeemPrepare(prepareParams)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	preparedEvs = append(preparedEvs, &RedeemPreparedEvent{
		RedeemPrepareTxHash: [32]byte(tx.Hash().Bytes()),
		RedeemRequestTxHash: prepareParams.RedeemRequestTxHash,
		Amount:              new(big.Int).Set(prepareParams.Amount),
		Requester:           prepareParams.Requester,
		Receiver:            string(prepareParams.Receiver),
		OutpointTxIds:       prepareParams.OutpointTxIds,
		OutpointIdxs:        prepareParams.OutpointIdxs,
	})

	err = env.Etherman.TWBTCApprove(env.Sim.Accounts[1], big.NewInt(100))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	requestParams := env.GenRequestParams(&etherman.ParamConfig{Sender: 1, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	tx, err = env.Etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RedeemRequestTxHash: [32]byte(tx.Hash().Bytes()),
		Requester:           requestParams.Auth.From,
		Amount:              new(big.Int).Set(requestParams.Amount),
		Receiver:            string(requestParams.Receiver),
		IsValidReceiver:     true,
	})

	requestParams = env.GenRequestParams(&etherman.ParamConfig{Sender: 1, Amount: big.NewInt(20)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	// set invalid btc address
	requestParams.Receiver = "abcd"
	tx, err = env.Etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Sim.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RedeemRequestTxHash: [32]byte(tx.Hash().Bytes()),
		Requester:           requestParams.Auth.From,
		Amount:              new(big.Int).Set(requestParams.Amount),
		Receiver:            requestParams.Receiver,
		IsValidReceiver:     false,
	})

	return
}
