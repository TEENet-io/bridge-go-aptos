package main

// 铸币参数
type MintParams struct {
	BtcTxId   []byte // 比特币交易哈希
	Amount    uint64 // 铸币金额
	Receiver  string // 接收者Aptos地址
	Signature []byte // 签名数据
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
	Signature     []byte   // 签名数据
}

// 铸币事件
type MintedEvent struct {
	TxVersion string // Aptos交易版本号
	BtcTxId   string // 比特币交易ID
	Receiver  string // 接收者Aptos地址
	Amount    uint64 // 金额
}

// 赎回请求事件
type RedeemRequestEvent struct {
	TxVersion string // Aptos交易版本号
	Sender    string // 发送者Aptos地址
	Receiver  string // 接收者比特币地址
	Amount    uint64 // 金额
}

// 赎回准备事件
type RedeemPreparedEvent struct {
	TxVersion     string   // Aptos交易版本号
	EthTxHash     string   // 请求交易哈希
	Requester     string   // 请求者地址
	Receiver      string   // 接收者比特币地址
	Amount        uint64   // 金额
	OutpointTxIds []string // 比特币UTXO交易ID列表
	OutpointIdxs  []uint16 // 对应的输出索引
}
