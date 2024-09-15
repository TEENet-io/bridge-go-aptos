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
	assert.True(t, redeem.Equal(clone))
}

func TestHasPrepared(t *testing.T) {
	redeem := &Redeem{}
	assert.False(t, redeem.HasPrepared())

	redeem.PrepareTxHash = common.RandBytes32()
	assert.True(t, redeem.HasPrepared())
}

func TestHasCompleted(t *testing.T) {
	redeem := &Redeem{}
	assert.False(t, redeem.HasCompleted())

	redeem.BtcTxId = common.RandBytes32()
	assert.True(t, redeem.HasCompleted())
}

func TestSet(t *testing.T) {
	redeem := &Redeem{}
	reqEv := &ethsync.RedeemRequestedEvent{}
	reqEv.RedeemRequestTxHash = common.RandBytes32()
	reqEv.Requester = common.RandEthAddress()
	reqEv.Amount = big.NewInt(100)
	reqEv.Receiver = "abcd"

	redeem.SetFromRequestedEvent(reqEv)
	assert.Equal(t, reqEv.RedeemRequestTxHash, redeem.RequestTxHash)
	assert.Equal(t, reqEv.Requester, redeem.Requester)
	assert.Equal(t, reqEv.Amount, redeem.Amount)
	assert.Equal(t, reqEv.Receiver, redeem.Receiver)
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
