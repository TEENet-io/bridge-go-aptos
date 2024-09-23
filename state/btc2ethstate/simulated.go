package btc2ethstate

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
)

func RandMint(amount int, numOfOutpoints int, status MintStatus) *Mint {
	mint := &Mint{
		BtcTxID:    common.RandBytes32(),
		MintTxHash: common.RandBytes32(),
		Receiver:   common.RandEthAddress(),
		Amount:     big.NewInt(int64(amount)),
		Status:     status,
	}

	mint.Outpoints = []state.Outpoint{}
	for i := 0; i < numOfOutpoints; i++ {
		mint.Outpoints = append(mint.Outpoints, state.Outpoint{
			TxId: common.RandBytes32(),
			Idx:  uint16(i),
		})
	}

	return mint
}
