package state

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
)

func randRedeem(status RedeemStatus) *Redeem {
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
		Receiver: "rand_btc_address",
		Status:   status,
	}
}
