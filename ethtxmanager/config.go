package ethtxmanager

import "time"

type Config struct {
	FrequencyToGetRequestedRedeems time.Duration
	FrequencyToGetSignatures       time.Duration
}
