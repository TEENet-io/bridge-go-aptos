package btcvault

// VaultUTXO represents an unspent transaction output
type VaultUTXO struct {
	BlockNumber int32  // Block number (height)
	BlockHash   string // 64-character hexadecimal string (no 0x prefix)
	TxID        string // 64-character hexadecimal string (no 0x prefix)
	Vout        int32  // Output index
	Amount      int64  // Amount in satoshis
	PkScript    []byte // Public key script (shall use when unlocking this script)
	Lockup      bool   // Lockup status, default is false
	Spent       bool   // Spent status, default is false
	Timeout     int64  // Unix timestamp in seconds, set to 0 if untouched
}

// VaultUTXOStorage defines the interface for database operations on VaultUTXO
type VaultUTXOStorage interface {
	// InsertVaultUTXO inserts a new VaultUTXO into the database
	InsertVaultUTXO(utxo VaultUTXO) error

	// Select all UTXOs that are usable (not locked, not spent)
	QueryAllUsableUTXOs() ([]VaultUTXO, error)

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
	// Excludes locked UTXOs.
	// Excludes spent UTXOs.
	SumMoney() (int64, error)
}
