package common

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func IsValidBtcAddress(address string, cfg *chaincfg.Params) bool {
	if _, err := btcutil.DecodeAddress(address, cfg); err != nil {
		return false
	}

	return true
}

func MainNetParams() *chaincfg.Params {
	return &chaincfg.MainNetParams
}
