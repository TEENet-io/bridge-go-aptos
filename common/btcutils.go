package common

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
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

// TODO: rx and s are both 32 bytes, or just pass in b as 64 bytes
func Verify(pub, msg []byte, rx, s *big.Int) bool {
	// fill (rx, s) => b
	// [32]byte + [32]byte => [64]byte
	b := []byte{}
	RX := BigInt2Bytes32(rx)
	S := BigInt2Bytes32(s)
	b = append(b, RX[:]...)
	b = append(b, S[:]...)

	pub = append([]byte{0x02}, pub...)
	pubKey, err := btcec.ParsePubKey(pub)
	if err != nil {
		return false
	}

	// b (64byte) => signature object
	sig, err := schnorr.ParseSignature(b[:])
	if err != nil {
		return false
	}

	return sig.Verify(msg, pubKey)
}
