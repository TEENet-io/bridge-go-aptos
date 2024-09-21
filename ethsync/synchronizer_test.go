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
	common.EthStartingBlock = big.NewInt(10)

	common.Debug = true
	logger.Debug("DEBUG MODE ON")
	defer func() {
		logger.Debug("DEBUG MODE OFF")
		common.Debug = false
	}()

	env, err := etherman.NewSimEtherman()
	assert.NoError(t, err)

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

	// No event should be sent since the finalized block number is too small
	ctx1, cancel1 := context.WithCancel(context.Background())
	go mockB2EState.Start(ctx1)
	go mockE2BState.Start(ctx1)
	go synchronizer.Sync(ctx1)
	sendTxs(t, env)
	time.Sleep(500 * time.Millisecond)
	cancel1()
	assert.Empty(t, mockB2EState.mintedEv)
	assert.Empty(t, mockE2BState.requestedEv)
	assert.Empty(t, mockE2BState.preparedEv)

	// test when the finalized block number is valid
	ctx2, cancel2 := context.WithCancel(context.Background())
	blk, _ := env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	start := blk.Number()
	assert.NoError(t, err)
	for start.Cmp(synchronizer.lastFinalizedBlockNumber) != 1 {
		env.Chain.Backend.Commit()
		start.Add(start, big.NewInt(1))
	}
	blk, _ = env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.Equal(t, blk.Number(),
		synchronizer.lastFinalizedBlockNumber.Add(synchronizer.lastFinalizedBlockNumber, big.NewInt(1)))

	go mockB2EState.Start(ctx2)
	go mockE2BState.Start(ctx2)
	go synchronizer.Sync(ctx2)

	mintedEvs, reqeustedEvs, preparedEvs := sendTxs(t, env)
	time.Sleep(200 * time.Millisecond)
	cancel2()

	blk, _ = env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.Equal(t, blk.Number(), mockE2BState.lastFinalized)
	assert.Equal(t, 2, len(mockE2BState.requestedEv))
	assert.Equal(t, 1, len(mockE2BState.preparedEv))
	assert.Equal(t, 1, len(mockB2EState.mintedEv))

	assert.Equal(t, mintedEvs[0], mockB2EState.mintedEv[0])
	assert.Equal(t, reqeustedEvs[0], mockE2BState.requestedEv[0])
	assert.Equal(t, reqeustedEvs[1], mockE2BState.requestedEv[1])
	assert.Equal(t, preparedEvs[0], mockE2BState.preparedEv[0])
}

func sendTxs(t *testing.T, env *etherman.SimEtherman) (
	mintedEvs []*MintedEvent,
	requestedEvs []*RedeemRequestedEvent,
	preparedEvs []*RedeemPreparedEvent,
) {
	mintParams := env.GenMintParams(&etherman.ParamConfig{Receiver: 1, Amount: big.NewInt(100)})
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

	prepareParams := env.GenPrepareParams(
		&etherman.ParamConfig{Requester: 4, Amount: big.NewInt(400), OutpointNum: 1})
	tx, err = env.Etherman.RedeemPrepare(prepareParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	preparedEvs = append(preparedEvs, &RedeemPreparedEvent{
		PrepareTxHash: [32]byte(tx.Hash().Bytes()),
		RequestTxHash: prepareParams.RequestTxHash,
		Amount:        new(big.Int).Set(prepareParams.Amount),
		Requester:     prepareParams.Requester,
		Receiver:      string(prepareParams.Receiver),
		OutpointTxIds: prepareParams.OutpointTxIds,
		OutpointIdxs:  prepareParams.OutpointIdxs,
	})

	err = env.Etherman.TWBTCApprove(env.Chain.Accounts[1], big.NewInt(100))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	requestParams := env.GenRequestParams(&etherman.ParamConfig{Requester: 1, Amount: big.NewInt(80)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	tx, err = env.Etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RequestTxHash:   [32]byte(tx.Hash().Bytes()),
		Requester:       requestParams.Auth.From,
		Amount:          new(big.Int).Set(requestParams.Amount),
		Receiver:        string(requestParams.Receiver),
		IsValidReceiver: true,
	})

	requestParams = env.GenRequestParams(&etherman.ParamConfig{Requester: 1, Amount: big.NewInt(20)})
	if requestParams == nil {
		t.Fatal("failed to generate request params")
	}
	// set invalid btc address
	requestParams.Receiver = "abcd"
	tx, err = env.Etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RequestTxHash:   [32]byte(tx.Hash().Bytes()),
		Requester:       requestParams.Auth.From,
		Amount:          new(big.Int).Set(requestParams.Amount),
		Receiver:        requestParams.Receiver,
		IsValidReceiver: false,
	})

	return
}
