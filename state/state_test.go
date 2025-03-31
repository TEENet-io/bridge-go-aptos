package state

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/stretchr/testify/assert"
)

func newTestStateEnv(t *testing.T) (
	st *State,
	ctx context.Context,
	cancel context.CancelFunc,
	close func(),
) {
	sqlDB := getMemoryDB()

	statedb, err := NewStateDB(sqlDB)
	assert.NoError(t, err)

	st, err = New(statedb, &StateConfig{ChannelSize: 1, UniqueChainId: big.NewInt(1337)})
	assert.NoError(t, err)

	ctx, cancel = context.WithCancel(context.Background())

	close = func() {
		statedb.Close()
		sqlDB.Close()
	}

	return
}

func TestInitChainId(t *testing.T) {
	st, _, cancel, close := newTestStateEnv(t)
	defer close()
	defer cancel()

	b := st.cache.ethChainId.Load().([]byte)
	assert.NotNil(t, b)
	chainId := new(big.Int).SetBytes(b)
	assert.Equal(t, chainId, big.NewInt(1337))

	bs, ok, err := st.statedb.GetKeyedValue(KeyEthChainId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, chainId, new(big.Int).SetBytes(bs[:]))
}

func TestUnmatchedChainId(t *testing.T) {
	sqlDB := getMemoryDB()

	statedb, err := NewStateDB(sqlDB)
	assert.NoError(t, err)
	err = statedb.SetKeyedValue(KeyEthChainId, common.BigInt2Bytes32(big.NewInt(1338)))
	assert.NoError(t, err)

	st, err := New(statedb, &StateConfig{ChannelSize: 1, UniqueChainId: big.NewInt(1337)})
	assert.Equal(t, err, ErrEthChainIdUnmatchedStored)
	assert.Nil(t, st)
}

func TestNewStateWithoutStored(t *testing.T) {
	sqlDB := getMemoryDB()
	statedb, _ := NewStateDB(sqlDB)
	defer sqlDB.Close()
	defer statedb.Close()

	_, ok, err := statedb.GetKeyedValue(KeyEthFinalizedBlock)
	assert.NoError(t, err)
	assert.False(t, ok)

	st, err := New(statedb, &StateConfig{ChannelSize: 1})
	assert.NoError(t, err)

	finalized, err := st.GetEthFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, finalized, common.EthStartingBlock)
}

func TestErrStoredEthFinalizedBlockNumberInvalid(t *testing.T) {
	sqlDB := getMemoryDB()
	statedb, _ := NewStateDB(sqlDB)
	defer sqlDB.Close()
	defer statedb.Close()

	// set stored = default - 1
	stored := new(big.Int).Sub(common.EthStartingBlock, big.NewInt(1))
	err := statedb.SetKeyedValue(KeyEthFinalizedBlock, common.BigInt2Bytes32(stored))
	assert.NoError(t, err)

	_, err = New(statedb, &StateConfig{ChannelSize: 1})
	assert.Equal(t, err, ErrStoredEthFinalizedBlockNumberInvalid)
}

func TestWhenNewFBNLessThanStored(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer close()
	defer cancel()

	stored, err := st.GetEthFinalizedBlockNumber()
	assert.NoError(t, err)

	go st.Start(ctx)

	// no change when the new finalized block number is equal or less than the stored one
	minusOne := new(big.Int).Sub(stored, big.NewInt(1))
	st.GetNewEthFinalizedBlockChannel() <- minusOne
	time.Sleep(100 * time.Millisecond)
	curr, err := st.GetEthFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, curr, stored)
}

func TestWhenNewFBNLargerThanStored(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer close()
	defer cancel()

	stored, err := st.GetEthFinalizedBlockNumber()
	assert.NoError(t, err)

	go st.Start(ctx)

	// new = stored + 1
	plusOne := new(big.Int).Add(stored, big.NewInt(1))
	st.GetNewEthFinalizedBlockChannel() <- plusOne
	time.Sleep(100 * time.Millisecond)
	curr, err := st.GetEthFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, curr, plusOne)
}

func TestErrRequestedEventInvalid(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer cancel()
	defer close()

	ch := st.GetNewRedeemRequestedEventChannel()

	var err error

	ev := &ethsync.RedeemRequestedEvent{}

	// empty requestTxHash
	go func() { err = st.Start(ctx) }()
	ch <- ev
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, err.Error(), ErrRequestedEventInvalid.Error())
	ev.RequestTxHash = common.RandBytes32()

	// empty requester
	go func() { err = st.Start(ctx) }()
	ch <- ev
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, err.Error(), ErrRequestedEventInvalid.Error())
	ev.Requester = common.RandEthAddress()

	// invalid amount
	go func() { err = st.Start(ctx) }()
	ch <- ev
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, err.Error(), ErrRequestedEventInvalid.Error())
}

func TestNewRedeemRequestedEvent(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer cancel()
	defer close()

	ch := st.GetNewRedeemRequestedEventChannel()

	// generate a random event and generate a redeem from it for comparison
	ev := ethsync.RandRedeemRequestedEvent(100, true)
	expected, err := createRedeemFromRequestedEvent(ev)
	assert.NoError(t, err)

	go func() { err = st.Start(ctx) }()

	ch <- ev
	time.Sleep(100 * time.Millisecond)

	actual, ok, err := st.statedb.GetRedeem(ev.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	// Change the amount and send the event again. Since the requestTxHash
	// remains the same, the new event would not do anything.
	ev.Amount = new(big.Int).Add(ev.Amount, big.NewInt(1))
	ch <- ev
	time.Sleep(100 * time.Millisecond)
	actual, ok, err = st.statedb.GetRedeem(ev.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)
}

func TestErrUpdateInvalidRedeem(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer cancel()
	defer close()

	reqCh := st.GetNewRedeemRequestedEventChannel()
	prepCh := st.GetNewRedeemPreparedEventChannel()

	reqEv := ethsync.RandRedeemRequestedEvent(100, false)
	prepEv := &ethsync.RedeemPreparedEvent{
		RequestTxHash: reqEv.RequestTxHash,
	}

	var err error
	go func() {
		err = st.Start(ctx)
	}()

	// register invalid redeem
	reqCh <- reqEv
	time.Sleep(100 * time.Millisecond)

	// send the prepared event with the same requestTxHash
	prepCh <- prepEv
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, err, ErrUpdateInvalidRedeem)
}

func TestNewRedeemPreparedEvent(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer cancel()
	defer close()

	ch1 := st.GetNewRedeemRequestedEventChannel()
	ch2 := st.GetNewRedeemPreparedEventChannel()

	ev1 := ethsync.RandRedeemPreparedEvent(100, 2)      //generate a random prepared event
	expected, err := createRedeemFromPreparedEvent(ev1) // generate a redeem from the event for comparison
	assert.NoError(t, err)

	go func() { st.Start(ctx) }()

	// insert without a corresponding request redeem stored
	ch2 <- ev1                         // send the prepared event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	actual, ok, err := st.statedb.GetRedeem(ev1.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	// update with a corresponding request redeem stored
	ev2 := ethsync.RandRedeemPreparedEvent(200, 1)
	ev3 := &ethsync.RedeemRequestedEvent{
		RequestTxHash:   ev2.RequestTxHash,
		Requester:       ev2.Requester,
		Amount:          ev2.Amount,
		Receiver:        ev2.Receiver,
		IsValidReceiver: true,
	}
	ch1 <- ev3                         // insert the requested event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	ch2 <- ev2                         // send the prepared event
	time.Sleep(100 * time.Millisecond) // wait for the state to process the event
	expected, err = createRedeemFromPreparedEvent(ev2)
	assert.NoError(t, err)
	actual, ok, err = st.statedb.GetRedeem(ev2.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)

	cancel()
}

func TestNewMintedEvent(t *testing.T) {
	st, ctx, cancel, close := newTestStateEnv(t)
	defer cancel()
	defer close()

	ch := st.GetNewMintedEventChannel()

	ev := ethsync.RandMintedEvent(100)
	expected := createMintFromMintedEvent(ev)

	go func() { st.Start(ctx) }()

	ch <- ev
	time.Sleep(100 * time.Millisecond)

	actual, ok, err := st.statedb.GetMint(ev.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected, actual)
}
