package ethtxmanager

import "time"

type EthTxMgrConfig struct {
	// Frequency to get all the requested redeems that have not been prepared
	IntervalToPrepareRedeem time.Duration

	// Frequency to monitor pending transactions
	IntervalToMonitorPendingTxs time.Duration

	IntervalToMint time.Duration

	// Timeout on waiting for a schnorr threshold signature
	TimeoutOnWaitingForSignature time.Duration

	// Timeout on waiting for the spendable outpoints from BTC wallet
	TimeoutOnWaitingForOutpoints time.Duration

	// Timeout on waiting for the "monitored Tx" to be mined
	TimeoutOnMonitoringPendingTxs uint64
}
