package eth2btcstate

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	redeem := randRedeem()
	jRedeem := &JSONRedeem{
		RequestTxHash: common.Bytes32ToHexStr(redeem.RequestTxHash),
		PrepareTxHash: common.Bytes32ToHexStr(redeem.PrepareTxHash),
		BtcTxId:       common.Bytes32ToHexStr(redeem.BtcTxId),
		Requester:     redeem.Requester.Hex(),
		Amount:        common.BigIntToHexStr(redeem.Amount),
		Outpoints: []JSONOutpoint{
			{
				TxId: common.Bytes32ToHexStr(redeem.Outpoints[0].TxId),
				Idx:  redeem.Outpoints[0].Idx,
			},
			{
				TxId: common.Bytes32ToHexStr(redeem.Outpoints[1].TxId),
				Idx:  redeem.Outpoints[1].Idx,
			},
		},
		Receiver: redeem.Receiver,
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
	redeem := randRedeem()

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
	redeem := &Redeem{}

	_, err := redeem.SetFromRequestedEvent(ev)
	assert.Equal(t, ErrorRedeemRequestTxHashInvalid, err.Error())

	// invalid requester
	ev.RedeemRequestTxHash = common.RandBytes32()
	_, err = redeem.SetFromRequestedEvent(ev)
	assert.Equal(t, ErrorRequesterInvalid, err.Error())

	// nil amount
	ev.Requester = common.RandEthAddress()
	_, err = redeem.SetFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// zero amount
	ev.Amount = big.NewInt(0)
	_, err = redeem.SetFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// success
	ev.Amount = big.NewInt(100)
	ev.Receiver = "valid_btc_address"
	ev.IsValidReceiver = true

	_, err = redeem.SetFromRequestedEvent(ev)
	assert.NoError(t, err)
}

func TestUpdateFromPrepareEvent(t *testing.T) {
	redeem := randRedeem()
	ev := &ethsync.RedeemPreparedEvent{}

	// invalid request tx hash
	_, err := redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorRedeemPrepareTxHashInvalid, err.Error())

	// unmatch request tx hash
	ev.RedeemPrepareTxHash = common.RandBytes32()
	ev.RedeemRequestTxHash = common.RandBytes32()
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorRedeemRequestTxHashUnmatched, err.Error())

	// unmatched requester
	ev.RedeemRequestTxHash = redeem.RequestTxHash
	ev.Requester = common.RandEthAddress()
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorRequesterUnmatched, err.Error())

	// nil amount
	ev.Requester = redeem.Requester
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid, err.Error())

	// unmatch amount
	ev.Amount = new(big.Int).Add(redeem.Amount, big.NewInt(1))
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorAmountUnmatched, err.Error())

	// receiver unmatched
	ev.Amount = new(big.Int).Set(redeem.Amount)
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorReceiverUnmatched, err.Error())

	// empty outpoints
	ev.Receiver = redeem.Receiver
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorOutpointsEmpty, err.Error())
	ev.OutpointTxIds = [][32]byte{common.RandBytes32()}
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.Equal(t, ErrorOutpointsEmpty, err.Error())

	// success
	ev.OutpointIdxs = []uint16{0}
	_, err = redeem.SetFromPreparedEvent(ev)
	assert.NoError(t, err)
}

func randRedeem() *Redeem {
	return &Redeem{
		RequestTxHash: common.RandBytes32(),
		PrepareTxHash: common.RandBytes32(),
		BtcTxId:       common.RandBytes32(),
		Requester:     common.RandEthAddress(),
		Amount:        big.NewInt(100),
		Outpoints: []Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  0,
			},
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			},
		},
		Receiver: "abcd",
	}
}
