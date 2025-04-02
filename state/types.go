package state

import (
	"github.com/TEENet-io/bridge-go/agreement"
)

type JSONRedeem struct {
	RequestTxHash string                      `json:"request_txid"`
	PrepareTxHash string                      `json:"prepare_txid"`
	BtcTxId       string                      `json:"btc_txid"`
	Requester     string                      `json:"requester"` // include the 0x prefix
	Amount        string                      `json:"amount"`
	Outpoints     []agreement.JSONBtcOutpoint `json:"outpoints"`
	Receiver      string                      `json:"receiver"`
	Status        string                      `json:"status"`
}
