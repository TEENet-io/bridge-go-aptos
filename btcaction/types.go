package btcaction

// Basic is the information that should be included in all types of actions.
type Basic struct {
	BlockNumber int    // btc
	BlockHash   string // btc
	TxHash      string // TxHash is btc TxHash
}

// DepositAction is a struct that represents a deposit transaction
// from BTC to EVM.
// As the nature of BTC, inputs maybe several different UTXOs from different addresses.
// So we don't track inputs.
type DepositAction struct {
	Basic
	DepositValue    int64
	DepositReceiver string // of btc (our bridge wallet address)
	EvmID           int32  // EVM Chain ID
	EvmAddr         string // No 0x prefix
}

// DepositStorage is an interface for storing and querying DepositAction.
type DepositStorage interface {
	// AddDeposit adds a new DepositAction.
	AddDeposit(deposit DepositAction) error

	// GetDepositByTxHash queries DepositAction by TxHash.
	GetDepositByTxHash(txHash string) ([]DepositAction, error)

	// GetDepositByReceiver queries DepositAction by DepositReceiver.
	GetDepositByReceiver(receiver string) ([]DepositAction, error)

	// GetDepositByEVM queries DepositAction by EvmAddr and EvmID.
	GetDepositByEVM(evmAddr string, evmID int32) ([]DepositAction, error)

	GetDepositByEVMAddr(evmAddr string) ([]DepositAction, error)
}

// RedeemAction is a management action.
// After sent the redeem btc, we fill in EthRequestTxID, BtcHash, and mark Sent.
// After the redeem btc is mined, we mark Mined, and Basic
type RedeemAction struct {
	EthRequestTxID string // fill this after <sent>. 64 hex (32 byte), no "0x" prefix.
	BtcHash        string // BTC TxID. fill this after <sent>. no "0x" prefix.
	Sent           bool   // mark this after <sent>.
	Mined          bool   // mark this after <mined>.
}

type RedeemActionStorage interface {
	// Check if the redeem exists via ethRequestTxId ?
	HasRedeem(ethRequestTxID string) (bool, error)

	// Query a redeem by ethRequestTxId
	QueryByEthRequestTxId(ethRequestTxID string) (*RedeemAction, error)

	// Check via btcTxID
	QueryByBtcTxId(btcTxID string) (*RedeemAction, error)

	// Insert a new BTC redeem (sent to BTC blockchain but not mined yet)
	InsertRedeem(r *RedeemAction) error

	// Check if the redeem exists but not been mined on Bitcoin blockchain
	IfNotMined(ethRequestTxID string) (bool, error)

	// Complete (finish) the redeem
	CompleteRedeem(ethRequestTxID string) error

	// Complete (finish) the redeem by btcTxID
	// CompletedByBtcTxID(btcTxID string) error
}

// OtherTransferAction is a struct that represents an unknown transfer
// to us in BTC.
type OtherTransferAction struct {
	Basic
	Vout             int
	TransferValue    int64
	TransferReceiver string // on btc
}

// OtherTransferStorage is an interface for storing and querying OtherTransferAction.
type OtherTransferStorage interface {
	// AddOtherTransfer adds a new OtherTransferAction.
	AddOtherTransfer(transfer OtherTransferAction) error

	// GetOtherTransferByTxHash queries OtherTransferAction by TxHash.
	GetOtherTransferByTxHash(txHash string) ([]OtherTransferAction, error)

	// GetOtherTransferByReceiver queries OtherTransferAction by TransferReceiver.
	GetOtherTransferByReceiver(receiver string) ([]OtherTransferAction, error)
}

// // RefundAction is a struct that represents a refund transaction.
// type RefundAction struct {
// 	Basic
// 	Receiver       string // on btc
// 	RefTxHash      string // previous transaction hash
// }

// // RefundStorage is an interface for storing and querying RefundAction.
// type RefundStorage interface {
// 	// AddRefund adds a new RefundAction.
// 	AddRefund(refund RefundAction) error

// 	// GetRefundByTxHash queries RefundAction by TxHash.
// 	GetRefundByTxHash(txHash string) ([]RefundAction, error)

// 	// GetRefundByReceiver queries RefundAction by Receiver.
// 	GetRefundByReceiver(receiver string) ([]RefundAction, error)
// }

// // WithdrawAction is a struct that represents a withdraw transaction
// // from EVM to BTC.
// type WithdrawAction struct {
// 	Basic
// 	WithdrawValue    int64  // in satoshi
// 	WithdrawReceiver string // on btc
// 	ChangeValue      int64
// 	ChangeReceiver   string // on btc
// }
