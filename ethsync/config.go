package ethsync

import (
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type Config struct {
	FrequencyToCheckFinalizedBlock time.Duration
	BtcChainConfig                 *chaincfg.Params
}
