package agreement

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

// This interface defines how Tx manager interact with BTC wallet (from BTC side).
// What is the expected behavior of the btc wallet from btc side
type BtcWallet interface {
	// Request sends a request to the wallet to get outpoints for preparing
	// the redeem indexed by the tx hash and then return the outpoints via
	// the provided channel. The btc wallet should temporarily lock the
	// outpoints with a timeout. It should also monitor the RedeemPrepared
	// events emitted from the bridge for permanent locking.
	Request(
		reqTxId common.Hash, // eth requestTxHash
		amount *big.Int,
		ch chan<- []BtcOutpoint, // this channel receives a slice of outputs.
	) error
}
