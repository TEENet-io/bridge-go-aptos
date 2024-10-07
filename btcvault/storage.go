package btcvault

// VaultUTXO represents an unspent transaction output
type VaultUTXO struct {
	BlockNumber int32  // Block number (height)
	BlockHash   string // 64-character hexadecimal string
	TxID        string // 64-character hexadecimal string
	Vout        int32  // Output index
	Amount      int64  // Amount in satoshis
	Lockup      bool   // Lockup status, default is false
	Spent       bool   // Spent status, default is false
	Timeout     int64  // Unix timestamp in seconds, set to 0 if untouched
}

// VaultUTXOStorage defines the interface for database operations on VaultUTXO
type VaultUTXOStorage interface {
	// InsertVaultUTXO inserts a new VaultUTXO into the database
	InsertVaultUTXO(utxo VaultUTXO) error

	// QueryByBlockNumber retrieves all VaultUTXOs with the specified block number
	QueryByBlockNumber(blockNumber int32) ([]VaultUTXO, error)

	// QueryByBlockHash retrieves all VaultUTXOs with the specified block hash
	QueryByBlockHash(blockHash string) ([]VaultUTXO, error)

	// QueryByTxID retrieves all VaultUTXOs with the specified transaction ID
	QueryByTxID(txID string) ([]VaultUTXO, error)

	// QueryByTxIDAndVout retrieves a VaultUTXO with the specified transaction ID and vout
	QueryByTxIDAndVout(txID string, vout int32) (*VaultUTXO, error)

	// Query those utxos whose expired + lockup status is true
	QueryExpiredAndLockedUTXOs(t int64) ([]VaultUTXO, error)

	// Select enough UTXOs to cover the specified amount
	// The utxos must be unlocked and unspent
	QueryEnoughUTXOs(amount int64) ([]VaultUTXO, error)

	// SetLockup sets the lockup status of a VaultUTXO identified by txID and vout
	SetLockup(txID string, vout int32, lockup bool) error

	// SetSpent sets the spent status of a VaultUTXO identified by txID and vout
	SetSpent(txID string, vout int32, spent bool) error

	// SetTimeout sets the expiry timepoint of a VaultUTXO identified by txID and vout
	// It is a Unix timestamp in seconds
	// If time pass beyond the timeout specified timepoint, the UTXO lock will be considered as expired
	SetTimeout(txID string, vout int32, timeout int64) error

	// SumMoney calculates the total amount of all VaultUTXOs
	SumMoney() (int64, error)
}

// SpentUTXO represents a spent transaction output
type SpentUTXO struct {
	RelatedTxID string // related 64-character hexadecimal string
	RelatedVout int32  // related Output index
	BlockNumber int32  // Block number (height)
	BlockHash   string // 64-character hexadecimal string
	TxID        string // 64-character hexadecimal string
	Vin         int32  // Input index
}

// SpentUTXOStorage defines the interface for database operations on SpentUTXO
type SpentUTXOStorage interface {
	// InsertSpentUTXO inserts a new SpentUTXO into the database
	InsertSpentUTXO(spentUTXO SpentUTXO) error

	// QueryByTxIDAndVin retrieves a SpentUTXO with the specified transaction ID and vin
	QueryByTxIDAndVin(txID string, vin int32) (*SpentUTXO, error)

	// QueryByRelatedTxIDAndRelatedVout retrieves a SpentUTXO with the specified related transaction ID and related vout
	QueryByRelatedTxIDAndRelatedVout(relatedTxID string, relatedVout int32) (*SpentUTXO, error)

	// SumMoney calculates the total amount of all SpentUTXOs
	SumMoney() (int64, error)
}
