package ethsync

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// RedeemRequestedEvent is the event when user
// requests a redeem (EVM2BTC).
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

// RedeemPreparedEvent is the event when
// we have prepared the BTC redeem (BTC2EVM).
// The UTXO(s) are chosen.
// But it hasn't been sent out on BTC side yet.
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

// MintedEvent reqpresents when TWBTC is minted
// on ETH side.
type MintedEvent struct {
	MintTxHash common.Hash
	BtcTxId    common.Hash
	Receiver   common.Address
	Amount     *big.Int
}

func (ev *MintedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}
