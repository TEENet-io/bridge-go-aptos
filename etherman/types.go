package etherman

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type Bytes32Hex string
type AddressHex string

type MintParams struct {
	Auth     *bind.TransactOpts
	BtcTxId  Bytes32Hex
	Amount   *big.Int
	Receiver AddressHex
	Rx       Bytes32Hex
	S        Bytes32Hex
}

type BTCAddress string

type RequestParams struct {
	Auth     *bind.TransactOpts
	Amount   *big.Int
	Receiver BTCAddress
}

type PrepareParams struct {
	Auth          *bind.TransactOpts
	TxHash        Bytes32Hex
	Requester     AddressHex
	Amount        *big.Int
	OutpointTxIds []Bytes32Hex
	OutpointIdxs  []uint16
	Rx            string
	S             string
}
