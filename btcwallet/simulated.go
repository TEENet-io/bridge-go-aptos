package btcwallet

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
)

func RandSpendable(amount int, blk int, lock bool) *Spendable {
	return &Spendable{
		BtcTxId:     common.RandBytes32(),
		Idx:         uint16(common.RandBigInt(2).Uint64()),
		Amount:      big.NewInt(int64(amount)),
		BlockNumber: big.NewInt(int64(blk)),
		Lock:        lock,
	}
}
