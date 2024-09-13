package state

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/etherman"
)

type Eth2BtcState interface {
	GetLastEthFinalizedBlockNumberChannel() chan<- *big.Int
	GetRedeemRequestedEventChannel() chan<- *etherman.RedeemRequestedEvent
	GetRedeemPreparedEventChannel() chan<- *etherman.RedeemPreparedEvent
}

type Btc2EthState interface {
	GetMintedEventChannel() chan<- *etherman.MintedEvent
}

type JSONOutpoint struct {
	TxId string `json:"txid"`
	Idx  uint16 `json:"idx"`
}

type JSONRedeem struct {
	RequestTxHash string         `json:"request_txid"`
	PrepareTxHash string         `json:"prepare_txid"`
	BtcTxId       string         `json:"btc_txid"`
	Requester     string         `json:"requester"`
	Amount        string         `json:"amount"`
	Outpoints     []JSONOutpoint `json:"outpoints"`
	Receiver      string         `json:"receiver"`
}
