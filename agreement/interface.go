package agreement

import (
	"math/big"
)

// StateChannel is the interface that the core state should implement.
// Each call of following function retuns a channel that accept corresponding variable.
// (and core state shall store or act upon the variable when it is received)
type StateChannel interface {
	// Channel
	// on eth it is finalized block number, on aptos it is the ledger version number.
	// If Sync find new block, fill in this channel
	GetNewBlockChainFinalizedLedgerNumberChannel() chan<- *big.Int

	// Same for BTC
	GetNewBtcFinalizedBlockChannel() chan<- *big.Int

	// If you found new RedeemRequest Event, fill in this channel
	GetNewRedeemRequestedEventChannel() chan<- *RedeemRequestedEvent

	// If you found new RedeemPrepared Event, fill in this channel
	GetNewRedeemPreparedEventChannel() chan<- *RedeemPreparedEvent

	// If you found new Minted Event, fill in this channel
	GetNewMintedEventChannel() chan<- *MintedEvent

	// This is NOT a channel, it reads the Finalized Block Number from state.
	GetEthFinalizedBlockNumber() (*big.Int, error)
}
