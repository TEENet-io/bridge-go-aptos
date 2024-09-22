package ethsync

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type RedeemRequestedEvent struct {
	RequestTxHash   common.Hash
	Requester       common.Address
	Receiver        string
	Amount          *big.Int
	IsValidReceiver bool
}

type RedeemPreparedEvent struct {
	RequestTxHash common.Hash
	PrepareTxHash common.Hash
	Requester     common.Address
	Receiver      string
	Amount        *big.Int
	OutpointTxIds []common.Hash
	OutpointIdxs  []uint16
}

type MintedEvent struct {
	MintedTxHash common.Hash
	BtcTxId      common.Hash
	Receiver     common.Address
	Amount       *big.Int
}
