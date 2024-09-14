package eth2btcstate

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
