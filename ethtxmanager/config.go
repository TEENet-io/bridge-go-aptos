package ethtxmanager

import "time"

type Config struct {
	// Frequency to get all the requested redeems that have not been prepared
	FrequencyToGetUnpreparedRedeem time.Duration

	// Timeout on waiting for a schnorr threshold signature
	TimeoutOnWaitingForSignature time.Duration

	// Timeout on waiting for the spendable outpoints
	TimeoutOnWaitingForOutpoints time.Duration

	// Frequency to monitor pending transactions
	FrequencyToMonitorPendingTxs time.Duration

	TimeoutOnMonitoringPendingTxs uint64 // in blocks
}
