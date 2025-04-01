package state

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type BtcOutpoint struct {
	BtcTxId ethcommon.Hash
	BtcIdx  uint16
}

type JSONBtcOutpoint struct {
	BtcTxId string `json:"txid"`
	BtcIdx  uint16 `json:"idx"`
}

type JSONRedeem struct {
	RequestTxHash string            `json:"request_txid"`
	PrepareTxHash string            `json:"prepare_txid"`
	BtcTxId       string            `json:"btc_txid"`
	Requester     string            `json:"requester"` // include the 0x prefix
	Amount        string            `json:"amount"`
	Outpoints     []JSONBtcOutpoint `json:"outpoints"`
	Receiver      string            `json:"receiver"`
	Status        string            `json:"status"`
}
