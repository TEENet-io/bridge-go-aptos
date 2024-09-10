package etherman

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type Bytes32Hex string
type AddressHex string

type MintParams struct {
	Auth     *bind.TransactOpts
	BtcTxId  Bytes32Hex
	Amount   uint32
	Receiver AddressHex
	Rx       Bytes32Hex
	S        Bytes32Hex
}

type BTCAddress string

type RequestParams struct {
	Auth     *bind.TransactOpts
	Amount   uint32
	Receiver BTCAddress
}

type PrepareParams struct {
	Auth          *bind.TransactOpts
	TxHash        Bytes32Hex
	Requester     AddressHex
	Amount        uint32
	OutpointTxIds []Bytes32Hex
	OutpointIdxs  []uint16
	Rx            string
	S             string
}
