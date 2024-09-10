package etherman

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type MintParams struct {
	Auth     *bind.TransactOpts
	BtcTxId  string
	Amount   *big.Int
	Receiver string
	Rx       string
	S        string
}

type BTCAddress string

type RequestParams struct {
	Auth     *bind.TransactOpts
	Amount   int
	Receiver BTCAddress
}

type PrepareParams struct {
	Auth          *bind.TransactOpts
	TxHash        string
	Requester     string
	Amount        int
	OutpointTxIds []string
	OutpointIdxs  []uint16
	Rx            string
	S             string
}
