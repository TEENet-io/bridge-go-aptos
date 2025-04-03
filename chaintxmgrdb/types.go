package chaintxmgrdb

import (
	"math/big"
)

// MonitoredTx is the structure stores the status of a Tx that we monitor.
type MonitoredTx struct {
	TxIdentifier                []byte            // The Tx ID been tracked, this is the primary key. No duplication allowed!
	RefIdentifier               []byte            // Reference Identifier associated with this Tx
	SentBlockchainLedgerNumber  *big.Int          // default nil (unknown), The Tx is sent at this point (blocknumber/ledger number/timestamp)
	FoundBlockchainLedgerNumber *big.Int          // default nil (unknown), The Tx is found at this point (either success or reverted)
	TxStatus                    MonitoredTxStatus // See below
}

// Enum for the status of the tx submitted to the blockchain.
type MonitoredTxStatus string

const (
	MalForm  MonitoredTxStatus = "malform"  // mal-form Tx, cannot be accepted by blockchain.
	Limbo    MonitoredTxStatus = "limbo"    // sent, but not found anywhere.
	Pending  MonitoredTxStatus = "pending"  // pending in the blockchain's mempool, not executed, yet.
	Success  MonitoredTxStatus = "success"  // included in the blockchain ledger.
	Reverted MonitoredTxStatus = "reverted" // the wanted change that applies to blockchain didn't succeed.
	Reorg    MonitoredTxStatus = "reorg"    // blockchain re-orged (raely)
	Timeout  MonitoredTxStatus = "timeout"  // For too long a time, it is not success or reverted.
)

// Defines what the DB should do
// Regardless of the underlying implmentation
type ChainTxMgrDB interface {
	// Release the resource that db occupies, no error returned.
	Close()

	// Insert Monitored Tx into DB
	// error = 1) duplicate insertion (same TxIdentifier), 2) database error, etc ...
	InsertMonitoredTx(tx *MonitoredTx) error

	// Delete by Tx id
	DeleteMonitoredTxByTxHash(identifier []byte) error

	// Get one Tx by identifier,
	// result can be null (if not found)
	GetMonitoredTxByTxIdentifier(identifier []byte) (*MonitoredTx, error)

	// Get Tx(s) by reference identifier,
	// result can be empty slice (if not found)
	GetMonitoredTxByRefIdentifier(refIdentifier []byte) ([]*MonitoredTx, error)

	// Get Tx(s) by status
	GetMonitoredTxByStatus(status MonitoredTxStatus) ([]*MonitoredTx, error)

	// Update refIdentifier
	UpdateRef(identifier []byte, refIdentifier []byte) error

	// Update SentBlockchainLedgerNumber
	UpdateSent(identifier []byte, sentAt *big.Int) error

	// Update FoundBlockchainLedgerNumber
	UpdateFound(identifier []byte, foundAt *big.Int) error

	// Update Status
	UpdateStatus(identifier []byte, status MonitoredTx) error
}
