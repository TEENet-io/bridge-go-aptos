package ethsync

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type Config struct {
	FrequencyToCheckFinalizedBlock time.Duration
	BtcChainConfig                 *chaincfg.Params
	EthChainID                     *big.Int
}
