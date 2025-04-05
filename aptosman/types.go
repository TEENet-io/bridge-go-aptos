package aptosman

// 铸币参数
type MintParams struct {
	BtcTxId  []byte // 比特币交易哈希
	Amount   uint64 // 铸币金额
	Receiver string // 接收者Aptos地址
	// Rx        *big.Int
	// S         *big.Int
}

// 赎回请求参数
type RequestParams struct {
	Amount   uint64 // 赎回金额
	Receiver string // 比特币接收地址
}

// 赎回准备参数
type PrepareParams struct {
	RequestTxHash string   // Aptos交易版本号或哈希
	Requester     string   // 请求者Aptos地址
	Receiver      string   // 接收者比特币地址
	Amount        uint64   // 金额
	OutpointTxIds []string // 比特币UTXO交易ID列表
	OutpointIdxs  []uint16 // 对应的输出索引
	// Rx            *big.Int
	// S             *big.Int
}

// 铸币事件
type MintedEvent struct {
	MintTxHash string // Aptos交易版本号
	BtcTxId    string // 比特币交易ID
	Receiver   string // 接收者Aptos地址
	Amount     uint64 // 金额
}

// type RedeemRequestedEvent struct {
// 	RequestTxHash string // Aptos交易版本号
// 	Requester     string // 请求者Aptos地址
// 	Receiver      string // 接收者比特币地址
// 	Amount        uint64 // 金额
// }

// 赎回请求事件
type RedeemRequestedEvent struct {
	RequestTxHash   string // Aptos交易版本号
	Requester       string // 请求者Aptos地址
	Receiver        string // 接收者比特币地址
	Amount          uint64 // 金额
	IsValidReceiver bool   // 是否有效接收者
}

// 赎回准备事件
type RedeemPreparedEvent struct {
	RequestTxHash string
	PrepareTxHash string
	Requester     string
	Receiver      string
	Amount        uint64
	OutpointTxIds []string
	OutpointIdxs  []uint16
}
