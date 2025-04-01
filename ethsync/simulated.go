package ethsync

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
)

func RandRedeemRequestedEvent(amount int, valid bool) *agreement.RedeemRequestedEvent {
	if valid {
		return &agreement.RedeemRequestedEvent{
			RequestTxHash:   common.RandBytes32(),
			Requester:       common.RandEthAddress().Bytes(),
			Amount:          big.NewInt(int64(amount)),
			Receiver:        "valid_btc_address",
			IsValidReceiver: true,
		}
	}

	return &agreement.RedeemRequestedEvent{
		RequestTxHash:   common.RandBytes32(),
		Requester:       common.RandEthAddress().Bytes(),
		Amount:          big.NewInt(int64(amount)),
		Receiver:        "Invalid_btc_address",
		IsValidReceiver: false,
	}
}

func RandRedeemPreparedEvent(amount int, outpointNum int) *agreement.RedeemPreparedEvent {
	ev := &agreement.RedeemPreparedEvent{
		RequestTxHash: common.RandBytes32(),
		PrepareTxHash: common.RandBytes32(),
		Requester:     common.RandEthAddress().Bytes(),
		Receiver:      "valid_btc_address",
		Amount:        big.NewInt(int64(amount)),
	}

	for i := 0; i < outpointNum; i++ {
		ev.OutpointTxIds = append(ev.OutpointTxIds, common.RandBytes32())
		ev.OutpointIdxs = append(ev.OutpointIdxs, uint16(i))
	}

	return ev
}

func RandMintedEvent(amount int) *agreement.MintedEvent {
	return &agreement.MintedEvent{
		BtcTxId:    common.RandBytes32(),
		MintTxHash: common.RandBytes32(),
		Receiver:   common.RandEthAddress().Bytes(),
		Amount:     big.NewInt(int64(amount)),
	}
}
