package ethsync

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type Config struct {
	FrequencyToCheckEthFinalizedBlock time.Duration
	FrequencyToCheckBtcFinalizedBlock time.Duration
	BtcChainConfig                    *chaincfg.Params
	EthChainID                        *big.Int
}
