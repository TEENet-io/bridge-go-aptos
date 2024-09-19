package eth2btcstate

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/stretchr/testify/assert"
)

func TestNewState(t *testing.T) {
	db, err := NewStateDB("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.close()
	st, err := New(db, &Config{ChannelSize: 1, CacheSize: 1})
	assert.NoError(t, err)
	finalized, err := st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, common.EthStartingBlock)

	// return error when stored finalized block number less than the default starting block number
	finalized = new(big.Int).Sub(common.EthStartingBlock, big.NewInt(1))
	err = db.setKeyedValue(KeyLastFinalizedBlock, finalized.Bytes())
	assert.NoError(t, err)
	_, err = New(db, &Config{ChannelSize: 1, CacheSize: 1})
	assert.Equal(t, err, stateErrors.StoredFinalizedBlockNumberLessThanStartingBlockNumber(finalized))
}

func TestNewFinalizedBlockNumber(t *testing.T) {
	st, err := NewSimState(1, 1)
	assert.NoError(t, err)
	defer st.Close()

	stored, err := st.GetFinalizedBlockNumber()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go st.Start(ctx)

	// no change when the new finalized block number is equal or less than the stored one
	minusOne := new(big.Int).Sub(stored, big.NewInt(1))
	st.GetNewFinalizedBlockChannel() <- minusOne
	time.Sleep(100 * time.Millisecond)
	curr, err := st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, curr, stored)

	// update when the new finalized block number is larger than the current one
	plusOne := new(big.Int).Add(stored, big.NewInt(1))
	st.GetNewFinalizedBlockChannel() <- plusOne
	time.Sleep(100 * time.Millisecond)
	curr, err = st.GetFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, curr, plusOne)
}

func TestNewRedeemRequestedEvent(t *testing.T) {
	st, err := NewSimState(1, 1)
	assert.NoError(t, err)
	defer st.Close()

	ch := st.GetNewRedeemRequestedEventChannel()

	// generate a random event and generate a redeem from it,
	// for comparison
	simSync := &ethsync.SimSync{}
	ev := simSync.RandRedeemRequestedEvent(100, true)
	expected, err := createRedeemFromRequestedEvent(ev)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { err = st.Start(ctx) }()
	defer cancel()

	ch <- ev
	// wait for the state to process the event
	time.Sleep(100 * time.Millisecond)

	// change the amount and send the event again
	ev.Amount = new(big.Int).Add(ev.Amount, big.NewInt(1))
	ch <- ev
	time.Sleep(100 * time.Millisecond)

	actual, err := st.db.GetByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Len(t, actual, 1) // only one redeem should be saved
	assert.Equal(t, expected, actual[0])
}

func TestNewRedeemPreparedEvent(t *testing.T) {
	st, err := NewSimState(1, 1)
	assert.NoError(t, err)
	defer st.Close()

	ch1 := st.GetNewRedeemRequestedEventChannel()
	ch2 := st.GetNewRedeemPreparedEventChannel()

	simSync := &ethsync.SimSync{}
	ev1 := simSync.RandRedeemPreparedEvent(100, 2)      //generate a random prepared event
	expected, err := createRedeemFromPreparedEvent(ev1) // generate a redeem from the event for comparison
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var retErr error
	go func() { retErr = st.Start(ctx) }()
	defer cancel()

	// Insert without a corresponding request redeem stored
	ch2 <- ev1                         // send the prepared event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	actual, ok, err := st.db.Get(ev1.RedeemRequestTxHash[:], RedeemStatusPrepared)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	// Insert with a corresponding request redeem stored
	ev2 := simSync.RandRedeemPreparedEvent(200, 1)
	ev3 := &ethsync.RedeemRequestedEvent{
		RedeemRequestTxHash: ev2.RedeemRequestTxHash,
		Requester:           ev2.Requester,
		Amount:              ev2.Amount,
		Receiver:            ev2.Receiver,
		IsValidReceiver:     true,
	}
	ch1 <- ev3                         // insert the requested event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	ch2 <- ev2                         // send the prepared event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	expected, err = createRedeemFromPreparedEvent(ev2)
	assert.NoError(t, err)
	actual, ok, err = st.db.Get(ev2.RedeemRequestTxHash[:], RedeemStatusPrepared)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	// Error when updating a invalid redeem request
	ev4 := simSync.RandRedeemPreparedEvent(300, 1)
	ev5 := &ethsync.RedeemRequestedEvent{
		RedeemRequestTxHash: ev4.RedeemRequestTxHash,
		Requester:           ev4.Requester,
		Amount:              ev4.Amount,
		Receiver:            "invalid_btc_address",
		IsValidReceiver:     false,
	}
	ch1 <- ev5                         // insert the requested event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	ch2 <- ev4                         // send the prepared event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	assert.Equal(t, retErr, stateErrors.CannotPrepareDueToRedeemRequestInvalid(ev4.RedeemRequestTxHash[:]))
}
