package ethsync

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type EthSyncConfig struct {
	IntervalCheckBlockchain time.Duration
	BtcChainConfig          *chaincfg.Params // used for verify btc address correctness (in RedeemRequest)
	EthChainID              *big.Int
	EthRetroScanBlkNum      int64 // retro scan block, tell Sync() to scan from this block, -1 to honor the valude in state.
}
