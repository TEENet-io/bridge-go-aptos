package chaintxmgrdb

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
)

// MonitoredTx is the structure stores the status of a Tx that we monitor.
type MonitoredTx struct {
	TxIdentifier                []byte   // The Tx ID been tracked, this is the primary key. No duplication allowed!
	RefIdentifier               []byte   // Reference Identifier associated with this Tx
	SentBlockchainLedgerNumber  *big.Int // default nil (unknown), The Tx is sent at this point (blocknumber/ledger number/timestamp)
	FoundBlockchainLedgerNumber *big.Int // default nil (unknown), The Tx is found at this point (either success or reverted)
	TxStatus                    agreement.MonitoredTxStatus
}

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
	// You can feed in a list of statuses like ['limbo', 'reverted']
	GetMonitoredTxByStatus(status []agreement.MonitoredTxStatus) ([]*MonitoredTx, error)

	// Update refIdentifier
	UpdateRef(identifier []byte, refIdentifier []byte) error

	// Update SentBlockchainLedgerNumber
	UpdateSent(identifier []byte, sentAt *big.Int) error

	// Update FoundBlockchainLedgerNumber
	UpdateFound(identifier []byte, foundAt *big.Int) error

	// Update Status
	UpdateTxStatus(identifier []byte, status agreement.MonitoredTxStatus) error
}
