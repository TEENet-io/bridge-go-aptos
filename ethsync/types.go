package ethsync

import (
	"fmt"
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

func (ev *RedeemRequestedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
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

func (ev *RedeemPreparedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}

type MintedEvent struct {
	MintedTxHash common.Hash
	BtcTxId      common.Hash
	Receiver     common.Address
	Amount       *big.Int
}

func (ev *MintedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}
