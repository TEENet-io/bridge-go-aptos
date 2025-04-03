// Implement following interfaces to make the bridge work with your chain.
package chainsync

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
)

// Chain's Sync Worker, do the dirty job.
type SyncWorker interface {
	// The worker shall implment this function.
	// on ETH it is the finalized block number.
	// on aptos it is newest ledger version?
	GetNewestLedgerFinalizedNumber() (*big.Int, error)

	// Fetch Interested events from the blockchain.
	// Notice, the events shall ordered from old -> new.
	// Otherwise the bridge process will have logic bugs.
	GetTimeOrderedEvents(oldNum *big.Int, newNum *big.Int) ([]agreement.MintedEvent, []agreement.RedeemRequestedEvent, []agreement.RedeemPreparedEvent, error)
}
