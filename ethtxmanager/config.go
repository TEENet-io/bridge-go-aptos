package ethtxmanager

import "time"

type Config struct {
	// frequency to get all the requested redeems that have not been prepared
	FrequencyToGetUnpreparedRedeem time.Duration

	// Timeout on the go routine that waits for a schnorr threshold signature
	TimeoutOnWaitingForSignature time.Duration

	TimeoutOnWaitingForOutpoints time.Duration
}
