package aptossync

import (
	"math/big"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
)

type AptosSyncConfig struct {
	IntervalCheckBlockchain time.Duration
	BtcChainConfig          *chaincfg.Params
	AptosChainID            *big.Int // Aptos chain addresss
	AptosRetroScanBlkNum    int64
}
