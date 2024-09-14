package eth2btcstate

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/stretchr/testify/assert"
)

func TestNewState(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)
	finalized, err := st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, common.EthStartingBlock)

	finalized = big.NewInt(100)
	db = rawdb.NewMemoryDatabase()
	db.Put(KeyLastFinalizedBlock, finalized.Bytes())
	st, err = New(db)
	assert.NoError(t, err)
	finalized, err = st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, big.NewInt(100))
}

func TestUpdateFinalizedBlockNumber(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go st.Start(ctx)

	// test updating last finalized block number
	ch := st.GetLastEthFinalizedBlockNumberChannel()
	ch <- new(big.Int).Sub(common.EthStartingBlock, big.NewInt(1))
	time.Sleep(100 * time.Millisecond)

	finalized, err := st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, common.EthStartingBlock)

	ch <- new(big.Int).Add(common.EthStartingBlock, big.NewInt(1))
	time.Sleep(100 * time.Millisecond)

	finalized, err = st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, common.EthStartingBlock.Add(common.EthStartingBlock, big.NewInt(1)))

	cancel()
}

func TestDBOps(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	redeem := randRedeem()
	err = st.put(redeem)
	assert.NoError(t, err)

	ok, err := st.has(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = st.has(common.RandBytes32())
	assert.NoError(t, err)
	assert.False(t, ok)

	redeem2, err := st.get(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, redeem.Equal(redeem2))

	// make sure the returned is a copy, not a reference
	redeem2.BtcTxId = common.RandBytes32()
	redeem3, err := st.get(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, redeem.Equal(redeem3))
}

func TestUpdateFromRequestEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	ch := st.GetRequestedEventChannel()

	ev := &etherman.RedeemRequestedEvent{}
	ev.TxHash = common.RandBytes32()
	ev.Sender = common.RandAddress()
	ev.Amount = big.NewInt(100)
	ev.Receiver = "abcd"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { err = st.Start(ctx) }()

	ch <- ev

	time.Sleep(100 * time.Millisecond)

	ok, err := st.has(ev.TxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	redeem, err := st.get(ev.TxHash)
	assert.NoError(t, err)
	assert.Equal(t, ev.TxHash, redeem.RequestTxHash)
	assert.Equal(t, ev.Sender, redeem.Requester)
	assert.Equal(t, ev.Amount, redeem.Amount)
	assert.Equal(t, ev.Receiver, redeem.Receiver)

	ch <- ev // warning should be printed
}

func TestErrFromPrepareEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	prepCh := st.GetPreparedEventChannel()

	prepEv := &etherman.RedeemPreparedEvent{}
	prepEv.TxHash = common.RandBytes32()
	prepEv.EthTxHash = common.RandBytes32()
	prepEv.Requester = common.RandAddress()
	prepEv.Amount = big.NewInt(100)
	prepEv.OutpointTxIds = [][32]byte{common.RandBytes32()}
	prepEv.OutpointIdxs = []uint16{0}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { err = st.Start(ctx) }()
	prepCh <- prepEv
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, RedeemNotFound, err.Error())
}

func TestUpdateFromPrepareEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	reqCh := st.GetRequestedEventChannel()
	prepCh := st.GetPreparedEventChannel()

	reqEv := &etherman.RedeemRequestedEvent{}
	reqEv.TxHash = common.RandBytes32()
	reqEv.Sender = common.RandAddress()
	reqEv.Amount = big.NewInt(100)
	reqEv.Receiver = "abcd"

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() { err = st.Start(ctx) }()

	reqCh <- reqEv

	time.Sleep(100 * time.Millisecond)

	prepEv := &etherman.RedeemPreparedEvent{}
	prepEv.TxHash = common.RandBytes32()
	prepEv.EthTxHash = reqEv.TxHash
	prepEv.Requester = reqEv.Sender
	prepEv.Amount = new(big.Int).Set(reqEv.Amount)
	prepEv.OutpointTxIds = [][32]byte{common.RandBytes32()}
	prepEv.OutpointIdxs = []uint16{0}

	prepCh <- prepEv

	time.Sleep(100 * time.Millisecond)

	redeem, err := st.get(reqEv.TxHash)
	assert.NoError(t, err)
	assert.Equal(t, prepEv.TxHash, redeem.PrepareTxHash)
	for i, outpoint := range redeem.Outpoints {
		assert.Equal(t, prepEv.OutpointTxIds[i], outpoint.TxId)
		assert.Equal(t, prepEv.OutpointIdxs[i], outpoint.Idx)
	}

	prepCh <- prepEv // warning should be printed
}
