package btc2ethstate

type JSONOutpoint struct {
	TxId string `json:"txid"`
	Idx  uint16 `json:"idx"`
}

type JSONMint struct {
	BtcTxId    string         `json:"btc_txid"`
	MintTxHash string         `json:"mint_txid"`
	Receiver   string         `json:"receiver"`
	Amount     string         `json:"amount"`
	Outpoints  []JSONOutpoint `json:"outpoints"`
	Status     string         `json:"status"`
}
