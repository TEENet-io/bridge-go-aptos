// Implement following interfaces to make the bridge work with your chain.

package chaintxmgr

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
)

// Mgr's worker on chain, do the dirty job.
type MgrWorker interface {
	// Get the latest ledger number from chain (block number on eth, ledger version number on aptos)
	// This number marks the latest height (advancement) of blockchain.
	GetLatestLedgerNumber() (*big.Int, error)

	// Call the smart contract and verify if the mint is already minted on chain
	// The query uses the mint's BTC tx id (prevent double mint check)
	IsMinted(btcTxId [32]byte) (bool, error)

	// Call the actual mint() on smart contract on chain
	// Note: this function shall return the approximate ledger number when this tx is submitted to blockchain.
	// If ledger number field is really unknown, set to nil.
	// Return the (mint_tx_hash, sent_at_ledger_number, error)
	DoMint(mint *agreement.MintParameter) ([]byte, *big.Int, error)

	// Call the smart contract and verify if the redeem is already prepared on chain
	// The query uses the redeem's request tx id (prevent double prepare check)
	IsPrepared(requestTxId [32]byte) (bool, error)

	// Call the actual prepare() on smart contract on chain
	// Note: this function shall return the approximate ledger number when this tx is submitted to blockchain.
	// If ledger number field is really unknown, set to nil.
	// Return the (prepare_tx_hash, sent_at_ledger_number, error)
	DoPrepare(prepare *agreement.PrepareParameter) ([]byte, *big.Int, error)

	// Check Tx Status on Chain
	// Each transaction is to commit a change to blockchain,
	// Naturally, the status of the transaction can be 'success' or 'reverted'
	GetTxStatus(txId []byte) (agreement.MonitoredTxStatus, error)
}
