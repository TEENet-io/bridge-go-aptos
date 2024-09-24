package state

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	redeem := randRedeem(RedeemStatusCompleted)
	jOutpoints := []JSONOutpoint{}
	for _, outpoint := range redeem.Outpoints {
		jOutpoints = append(jOutpoints, JSONOutpoint{
			TxId: outpoint.TxId.String(),
			Idx:  outpoint.Idx,
		})
	}
	jRedeem := &JSONRedeem{
		RequestTxHash: redeem.RequestTxHash.String(),
		PrepareTxHash: redeem.PrepareTxHash.String(),
		BtcTxId:       redeem.BtcTxId.String(),
		Requester:     redeem.Requester.String(),
		Amount:        common.BigIntToHexStr(redeem.Amount),
		Outpoints:     jOutpoints,
		Receiver:      redeem.Receiver,
		Status:        string(redeem.Status),
	}

	data1, err := redeem.MarshalJSON()
	assert.NoError(t, err)
	data2, err := json.Marshal(jRedeem)
	assert.NoError(t, err)
	assert.JSONEq(t, string(data1), string(data2))

	redeem2 := &Redeem{}
	err = json.Unmarshal(data1, redeem2)
	assert.NoError(t, err)
	assert.Equal(t, redeem, redeem2)
}

func TestClone(t *testing.T) {
	redeem := randRedeem(RedeemStatusCompleted)

	clone := redeem.Clone()
	assert.Equal(t, redeem, clone)
}

func TestHasPrepared(t *testing.T) {
	redeem := &Redeem{}
	assert.False(t, redeem.HasPrepared())

	redeem.PrepareTxHash = common.RandBytes32()
	redeem.Status = RedeemStatusPrepared
	assert.True(t, redeem.HasPrepared())
}

func TestHasCompleted(t *testing.T) {
	redeem := &Redeem{}
	assert.False(t, redeem.HasCompleted())

	redeem.BtcTxId = common.RandBytes32()
	redeem.Status = RedeemStatusCompleted
	assert.True(t, redeem.HasCompleted())
}

func TestSetFromRequestEvent(t *testing.T) {
	ev := &ethsync.RedeemRequestedEvent{}

	_, err := createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorRequestTxHashInvalid, err.Error())

	// invalid requester
	ev.RequestTxHash = common.RandBytes32()
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorRequesterInvalid, err.Error())

	// nil amount
	ev.Requester = common.RandEthAddress()
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// zero amount
	ev.Amount = big.NewInt(0)
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// success
	ev.Amount = big.NewInt(100)
	ev.Receiver = "valid_btc_address"
	ev.IsValidReceiver = true

	_, err = createRedeemFromRequestedEvent(ev)
	assert.NoError(t, err)
}

func TestUpdateFromPrepareEvent(t *testing.T) {
	redeem := randRedeem(RedeemStatusRequested)
	ev := &ethsync.RedeemPreparedEvent{}

	// invalid request tx hash
	_, err := redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorRedeemPrepareTxHashInvalid, err.Error())

	// unmatch request tx hash
	ev.PrepareTxHash = common.RandBytes32()
	ev.RequestTxHash = common.RandBytes32()
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorRequestTxHashUnmatched, err.Error())

	// unmatched requester
	ev.RequestTxHash = redeem.RequestTxHash
	ev.Requester = common.RandEthAddress()
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorRequesterUnmatched, err.Error())

	// nil amount
	ev.Requester = redeem.Requester
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// unmatch amount
	ev.Amount = new(big.Int).Add(redeem.Amount, big.NewInt(1))
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorAmountUnmatched, err.Error())

	// receiver unmatched
	ev.Amount = new(big.Int).Set(redeem.Amount)
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorReceiverUnmatched, err.Error())

	// empty outpoints
	ev.Receiver = redeem.Receiver
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorOutpointsInvalid, err.Error())
	ev.OutpointTxIds = append([]ethcommon.Hash{}, common.RandBytes32())
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.Equal(t, ErrorOutpointsInvalid, err.Error())

	// success
	ev.OutpointIdxs = []uint16{0}
	_, err = redeem.updateFromPreparedEvent(ev)
	assert.NoError(t, err)
}
