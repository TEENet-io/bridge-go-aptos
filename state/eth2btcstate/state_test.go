package eth2btcstate

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
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

	// return error when stored finalized block number less than the default starting block number
	finalized = new(big.Int).Sub(common.EthStartingBlock, big.NewInt(1))
	db = rawdb.NewMemoryDatabase()
	db.Put(KeyLastFinalizedBlock, finalized.Bytes())
	_, err = New(db)
	assert.Equal(t, err.Error(), ErrorFinalizedBlockNumberInvalid)
}

func TestUpdateFinalizedBlockNumber(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go st.Start(ctx)
	defer cancel()

	// test updating last finalized block number
	ch := st.GetNewFinalizedBlockChannel()
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
	assert.Equal(t, redeem, redeem2)

	// make sure the returned is a copy, not a reference
	redeem2.BtcTxId = common.RandBytes32()
	redeem3, err := st.get(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Equal(t, redeem, redeem3)
}

func TestNewRedeemRequestedEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	ch := st.GetNewRedeemRequestedEventChannel()

	ev := &ethsync.RedeemRequestedEvent{}
	ev.RedeemRequestTxHash = common.RandBytes32()
	ev.Requester = common.RandEthAddress()
	ev.Amount = big.NewInt(100)
	ev.Receiver = "valid_btc_address"
	ev.IsValidReceiver = true

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = st.Start(ctx) }()
	defer cancel()

	ch <- ev

	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, err)
	redeem1, err := st.get(ev.RedeemRequestTxHash)
	redeem2 := &Redeem{}
	redeem2, err = redeem2.SetFromRequestedEvent(ev)
	assert.Equal(t, redeem1, redeem2)

	ev.Requester = common.RandEthAddress()
	ev.Amount = big.NewInt(200)
	ch <- ev // warning should be printed
	time.Sleep(100 * time.Millisecond)

	// redeem not changed
	redeem3, err := st.get(ev.RedeemRequestTxHash)
	assert.NoError(t, err)
	assert.Equal(t, redeem1, redeem3)
}

func TestNewRedeemPreparedEvent(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	st, err := New(db)
	assert.NoError(t, err)

	redeem := randRedeem()
	ch := st.GetNewRedeemPreparedEventChannel()

	ev := &ethsync.RedeemPreparedEvent{}
	ev.RedeemPrepareTxHash = common.RandBytes32()
	ev.RedeemRequestTxHash = redeem.RequestTxHash
	ev.Requester = redeem.Requester
	ev.Receiver = redeem.Receiver
	ev.Amount = new(big.Int).Set(redeem.Amount)
	ev.OutpointTxIds = [][32]byte{common.RandBytes32(), common.RandBytes32()}
	ev.OutpointIdxs = []uint16{1, 2}

	run := func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() { err = st.Start(ctx) }()
		defer cancel()
		ch <- ev
		time.Sleep(100 * time.Millisecond)
	}

	// redeem not found
	run()
	assert.Equal(t, ErrorRedeemNotFound, err.Error())

	// redeem invalid
	redeem.Status = RedeemStatusInvalid
	err = st.put(redeem)
	assert.NoError(t, err)
	run()
	assert.Equal(t, ErrorRedeemInvalid, err.Error())

	// do nothing
	redeem.Status = RedeemStatusCompleted
	err = st.put(redeem)
	assert.NoError(t, err)
	run()
	redeem2, err := st.get(redeem.RequestTxHash)
	assert.Equal(t, redeem, redeem2)

	// success
	redeem.Status = RedeemStatusRequested
	err = st.put(redeem)
	assert.NoError(t, err)
	run()
	redeem3, err := st.get(redeem.RequestTxHash)
	assert.NoError(t, err)
	assert.Equal(t, RedeemStatusPrepared, redeem3.Status)
	redeem4, err := redeem.SetFromPreparedEvent(ev)
	assert.NoError(t, err)
	assert.Equal(t, redeem3, redeem4)
}
