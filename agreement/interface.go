package agreement

import (
	"math/big"
)

// StateChannel is the interface that the "state" should implement.
// Each call of following function retuns a channel that accept corresponding variable.
// (and "state" shall act upon the variable when it is received)
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

	// This is <NOT> a channel, it reads the finalized block/ledger number from state.
	GetEthFinalizedBlockNumber() (*big.Int, error)
}

// This interface defines how Tx manager interact with BTC UTXO responder (from BTC side).
// This function query for enough UTXO(s) to satisfy the amount.
type BtcUTXOResponder interface {
	// Request sends a request to the responder to get outpoints for preparing
	// the redeem indexed by the tx hash and then return the outpoints via
	// the provided channel. The btc utxo responder should temporarily lock the
	// outpoints with a timeout.
	Request(
		reqTxId []byte, // the request Tx on other blockchain that associated with this request of UTXO(s)
		amount *big.Int,
		ch chan<- []BtcOutpoint, // this channel receives a slice of outputs.
	) error
}
