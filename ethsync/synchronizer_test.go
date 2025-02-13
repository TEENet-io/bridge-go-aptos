package ethsync

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/multisig"
	logger "github.com/sirupsen/logrus"
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

	ss, err := multisig.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("failed to create schnorr wallet: %v", err)
	}

	env, err := etherman.NewSimEtherman(etherman.GenPrivateKeys(10), ss, big.NewInt(1337))
	assert.NoError(t, err)

	chainID, err := env.Etherman.Client().ChainID(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, chainID, big.NewInt(1337))

	st := NewMockState()

	cfg := &Config{
		FrequencyToCheckEthFinalizedBlock: 500 * time.Millisecond,
		BtcChainConfig:                    common.MainNetParams(),
		EthChainID:                        chainID,
	}

	synchronizer, err := New(env.Etherman, st, cfg)
	assert.NoError(t, err)

	// No event should be sent since the finalized block number is too small
	ctx1, cancel1 := context.WithCancel(context.Background())
	go st.Start(ctx1)
	go synchronizer.Sync(ctx1)
	sendTxs(t, env)
	time.Sleep(500 * time.Millisecond)
	cancel1()
	assert.Empty(t, st.mintedEv)
	assert.Empty(t, st.requestedEv)
	assert.Empty(t, st.preparedEv)

	// test when the finalized block number is valid
	ctx2, cancel2 := context.WithCancel(context.Background())
	blk, _ := env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	start := blk.Number()
	assert.NoError(t, err)
	for start.Cmp(synchronizer.lastFinalized) != 1 {
		env.Chain.Backend.Commit()
		start.Add(start, big.NewInt(1))
	}
	blk, _ = env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.Equal(t, blk.Number(),
		synchronizer.lastFinalized.Add(synchronizer.lastFinalized, big.NewInt(1)))

	go st.Start(ctx2)
	go synchronizer.Sync(ctx2)

	mintedEvs, reqeustedEvs, preparedEvs := sendTxs(t, env)
	time.Sleep(200 * time.Millisecond)
	cancel2()

	blk, _ = env.Chain.Backend.Client().BlockByNumber(context.Background(), nil)
	assert.Equal(t, blk.Number(), st.lastEthFinalized)
	assert.Equal(t, 2, len(st.requestedEv))
	assert.Equal(t, 1, len(st.preparedEv))
	assert.Equal(t, 1, len(st.mintedEv))

	assert.Equal(t, mintedEvs[0].String(), st.mintedEv[0].String())
	assert.Equal(t, reqeustedEvs[0].String(), st.requestedEv[0].String())
	assert.Equal(t, reqeustedEvs[1].String(), st.requestedEv[1].String())
	assert.Equal(t, preparedEvs[0].String(), st.preparedEv[0].String())
}

// sendTxs sends the following txs:
// 1. Mint 100 TWBTC to account [1]
// 2. Prepare redeem for account [4]
// 3. Approve 100 TWBTC for account [1]
// 4. Request 80 TWBTC for account [1] with a valid btc address
// 5. Request 20 TWBTC for account [1] with an invalid btc address
func sendTxs(t *testing.T, env *etherman.SimEtherman) (
	mintedEvs []*MintedEvent,
	requestedEvs []*RedeemRequestedEvent,
	preparedEvs []*RedeemPreparedEvent,
) {
	// 1
	mintParams := env.GenMintParams(
		&etherman.ParamConfig{Receiver: 1, Amount: big.NewInt(100)},
		common.RandBytes32(),
	)
	tx, err := env.Etherman.Mint(mintParams)
	if err != nil {
		t.Fatal(err)
	}

	mintedEvs = append(mintedEvs, &MintedEvent{
		MintTxHash: tx.Hash(),
		BtcTxId:    mintParams.BtcTxId,
		Amount:     new(big.Int).Set(mintParams.Amount),
		Receiver:   mintParams.Receiver,
	})

	// 2
	prepareParams := env.GenPrepareParams(
		&etherman.ParamConfig{Requester: 4, Amount: big.NewInt(400), OutpointNum: 1})
	tx, err = env.Etherman.RedeemPrepare(prepareParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	preparedEvs = append(preparedEvs, &RedeemPreparedEvent{
		PrepareTxHash: tx.Hash(),
		RequestTxHash: prepareParams.RequestTxHash,
		Amount:        new(big.Int).Set(prepareParams.Amount),
		Requester:     prepareParams.Requester,
		Receiver:      string(prepareParams.Receiver),
		OutpointTxIds: prepareParams.OutpointTxIds,
		OutpointIdxs:  prepareParams.OutpointIdxs,
	})

	// 3
	_, err = env.Etherman.TWBTCApprove(env.GetAuth(1), big.NewInt(100))
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	// 4
	requestParams := env.GenRequestParams(&etherman.ParamConfig{Requester: 1, Amount: big.NewInt(80)})
	tx, err = env.Etherman.RedeemRequest(env.GetAuth(1), requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RequestTxHash:   tx.Hash(),
		Requester:       env.Chain.Accounts[1].From,
		Amount:          new(big.Int).Set(requestParams.Amount),
		Receiver:        string(requestParams.Receiver),
		IsValidReceiver: true,
	})

	// 5
	requestParams = env.GenRequestParams(&etherman.ParamConfig{Requester: 1, Amount: big.NewInt(20)})
	requestParams.Receiver = "invalid_btc_address"
	tx, err = env.Etherman.RedeemRequest(env.GetAuth(1), requestParams)
	assert.NoError(t, err)
	time.Sleep(200 * time.Millisecond)
	env.Chain.Backend.Commit()

	requestedEvs = append(requestedEvs, &RedeemRequestedEvent{
		RequestTxHash:   tx.Hash(),
		Requester:       env.Chain.Accounts[1].From,
		Amount:          new(big.Int).Set(requestParams.Amount),
		Receiver:        requestParams.Receiver,
		IsValidReceiver: false,
	})

	return
}
