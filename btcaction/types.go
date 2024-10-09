package btcaction

type Basic struct {
	BlockNumber int
	BlockHash   string
	TxHash      string
}

// DepositAction is a struct that represents a deposit transaction
// from BTC to EVM.
// As the nature of BTC, inputs maybe several different UTXOs from different addresses.
// So we don't track inputs.
type DepositAction struct {
	Basic
	DepositValue    int64
	DepositReceiver string // on btc (our wallet address)
	EvmID           int32
	EvmAddr         string // 0x... on EVM
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
}

// WithdrawAction is a struct that represents a withdraw transaction
// from EVM to BTC.
type WithdrawAction struct {
	Basic
	WithdrawValue    int64  // in satoshi
	WithdrawReceiver string // on btc
	ChangeValue      int64
	ChangeReceiver   string // on btc
}

// WithdrawStorage is an interface for storing and querying WithdrawAction.
type WithdrawStorage interface {
	// AddWithdraw adds a new WithdrawAction.
	AddWithdraw(withdraw WithdrawAction) error

	// GetWithdrawByTxHash queries WithdrawAction by TxHash.
	GetWithdrawByTxHash(txHash string) ([]WithdrawAction, error)

	// GetWithdrawByValue queries WithdrawAction by WithdrawValue.
	GetWithdrawByValue(value int64) ([]WithdrawAction, error)

	// GetWithdrawByReceiver queries WithdrawAction by WithdrawReceiver.
	GetWithdrawByReceiver(receiver string) ([]WithdrawAction, error)

	// GetWithdrawByChangeReceiver queries WithdrawAction by ChangeReceiver.
	GetWithdrawByChangeReceiver(receiver string) ([]WithdrawAction, error)
}

// UnknownTransferAction is a struct that represents an unknown transfer
// to us in BTC.
type UnknownTransferAction struct {
	Basic
	Vout             int
	TransferValue    int64
	TransferReceiver string // on btc
}

// UnknownTransferStorage is an interface for storing and querying UnknownTransferAction.
type UnknownTransferStorage interface {
	// AddUnknownTransfer adds a new UnknownTransferAction.
	AddUnknownTransfer(transfer UnknownTransferAction) error

	// GetUnknownTransferByTxHash queries UnknownTransferAction by TxHash.
	GetUnknownTransferByTxHash(txHash string) ([]UnknownTransferAction, error)

	// GetUnknownTransferByReceiver queries UnknownTransferAction by TransferReceiver.
	GetUnknownTransferByReceiver(receiver string) ([]UnknownTransferAction, error)
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
