package ethsync

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type EthSyncConfig struct {
	FrequencyToCheckEthFinalizedBlock time.Duration
	BtcChainConfig                    *chaincfg.Params // used for verify btc address correctness (in RedeemRequest)
	EthChainID                        *big.Int
	EthRetroScanBlkNum                int64
}
