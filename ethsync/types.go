package ethsync

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type RedeemRequestedEvent struct {
	RedeemRequestTxHash [32]byte
	Requester           common.Address
	Receiver            string
	Amount              *big.Int
	IsValidReceiver     bool
}

type RedeemPreparedEvent struct {
	RedeemRequestTxHash [32]byte
	RedeemPrepareTxHash [32]byte
	Requester           common.Address
	Receiver            string
	Amount              *big.Int
	OutpointTxIds       [][32]byte
	OutpointIdxs        []uint16
}

type MintedEvent struct {
	MintedTxHash [32]byte
	BtcTxId      [32]byte
	Receiver     common.Address
	Amount       *big.Int
}
