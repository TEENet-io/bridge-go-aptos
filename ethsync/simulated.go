package ethsync

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
)

func RandRedeemRequestedEvent(amount int, valid bool) *RedeemRequestedEvent {
	if valid {
		return &RedeemRequestedEvent{
			RequestTxHash:   common.RandBytes32(),
			Requester:       common.RandEthAddress(),
			Amount:          big.NewInt(int64(amount)),
			Receiver:        "valid_btc_address",
			IsValidReceiver: true,
		}
	}

	return &RedeemRequestedEvent{
		RequestTxHash:   common.RandBytes32(),
		Requester:       common.RandEthAddress(),
		Amount:          big.NewInt(int64(amount)),
		Receiver:        "Invalid_btc_address",
		IsValidReceiver: false,
	}
}

func RandRedeemPreparedEvent(amount int, outpointNum int) *RedeemPreparedEvent {
	ev := &RedeemPreparedEvent{
		RequestTxHash: common.RandBytes32(),
		PrepareTxHash: common.RandBytes32(),
		Requester:     common.RandEthAddress(),
		Receiver:      "valid_btc_address",
		Amount:        big.NewInt(int64(amount)),
	}

	for i := 0; i < outpointNum; i++ {
		ev.OutpointTxIds = append(ev.OutpointTxIds, common.RandBytes32())
		ev.OutpointIdxs = append(ev.OutpointIdxs, uint16(i))
	}

	return ev
}
