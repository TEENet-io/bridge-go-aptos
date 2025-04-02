package state

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	redeem := RandRedeem(RedeemStatusCompleted)
	jOutpoints := []agreement.JSONBtcOutpoint{}
	for _, outpoint := range redeem.Outpoints {
		jOutpoints = append(jOutpoints, agreement.JSONBtcOutpoint{
			BtcTxId: outpoint.BtcTxId.String(),
			BtcIdx:  outpoint.BtcIdx,
		})
	}
	jRedeem := &JSONRedeem{
		RequestTxHash: redeem.RequestTxHash.String(),
		PrepareTxHash: redeem.PrepareTxHash.String(),
		BtcTxId:       redeem.BtcTxId.String(),
		Requester:     common.Prepend0xPrefix(common.ByteSliceToPureHexStr(redeem.Requester)),
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
	redeem := RandRedeem(RedeemStatusCompleted)

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
	ev := &agreement.RedeemRequestedEvent{}

	_, err := createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorRequestTxHashInvalid.Error(), err.Error())

	// invalid requester
	ev.RequestTxHash = common.RandBytes32()
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorRequesterInvalid.Error(), err.Error())

	// nil amount
	ev.Requester = common.RandEthAddress().Bytes()
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid.Error(), err.Error())

	// zero amount
	ev.Amount = big.NewInt(0)
	_, err = createRedeemFromRequestedEvent(ev)
	assert.Equal(t, ErrorAmountInvalid.Error(), err.Error())

	// success
	ev.Amount = big.NewInt(100)
	ev.Receiver = "valid_btc_address"
	ev.IsValidReceiver = true

	_, err = createRedeemFromRequestedEvent(ev)
	assert.NoError(t, err)
}

func TestUpdateFromPreparedEvent(t *testing.T) {
	redeem := RandRedeem(RedeemStatusRequested)
	prepEv := &agreement.RedeemPreparedEvent{
		RequestTxHash: redeem.RequestTxHash,
	}

	// unmatched requester
	prepEv.Requester = common.RandEthAddress().Bytes()
	_, err := redeem.updateFromPreparedEvent(prepEv)
	assert.Equal(t, err, ErrorRequesterUnmatched)
	prepEv.Requester = redeem.Requester

	// unmatched amount
	prepEv.Amount = new(big.Int).Add(redeem.Amount, big.NewInt(1))
	_, err = redeem.updateFromPreparedEvent(prepEv)
	assert.Equal(t, err, ErrorAmountUnmatched)
	prepEv.Amount = common.BigIntClone(redeem.Amount)

	// unmatched receiver
	prepEv.Receiver = "invalid_btc_address"
	_, err = redeem.updateFromPreparedEvent(prepEv)
	assert.Equal(t, err, ErrorReceiverUnmatched)
	prepEv.Receiver = redeem.Receiver

	// empty prepareTxHash
	prepEv.PrepareTxHash = [32]byte{}
	_, err = redeem.updateFromPreparedEvent(prepEv)
	assert.Equal(t, err, ErrorPrepareTxHashEmpty)
	prepEv.PrepareTxHash = common.RandBytes32()

	// invalid outpoint tx id
	prepEv.OutpointTxIds = []ethcommon.Hash{
		common.RandBytes32(),
		[32]byte{},
	}
	prepEv.OutpointIdxs = []uint16{0, 1}
	_, err = redeem.updateFromPreparedEvent(prepEv)
	assert.Equal(t, err, ErrorOutpointTxIdInvalid)
	prepEv.OutpointTxIds = []ethcommon.Hash{
		common.RandBytes32(),
		common.RandBytes32(),
	}

	// pass
	_, err = redeem.updateFromPreparedEvent(prepEv)
	assert.NoError(t, err)
}
