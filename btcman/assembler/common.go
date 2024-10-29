package assembler

import (
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/chaincfg"
)

// DecodeWIF decodes a string private key to *btcutil.WIF
func DecodeWIF(privKeyStr string) (*btcutil.WIF, error) {
	decoded := base58.Decode(privKeyStr)
	if len(decoded) == 0 {
		return nil, errors.New("invalid private key string (cannot pass base58 decode)")
	}

	wif, err := btcutil.DecodeWIF(privKeyStr)
	if err != nil {
		return nil, err
	}

	return wif, nil
}

// Decode Address decodes a string address to btcutil.Address
func DecodeAddress(addressStr string, network *chaincfg.Params) (btcutil.Address, error) {
	address, err := btcutil.DecodeAddress(addressStr, network)
	if err != nil {
		return nil, err
	}
	return address, nil
}

func GetMainnetParams() *chaincfg.Params {
	return &chaincfg.MainNetParams
}

func GetTestnetParams() *chaincfg.Params {
	return &chaincfg.TestNet3Params
}

func GetRegtestParams() *chaincfg.Params {
	return &chaincfg.RegressionNetParams
}
