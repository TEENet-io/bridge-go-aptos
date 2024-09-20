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

func Sign(sk *btcec.PrivateKey, msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(sk, msg[:])
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}

func Verify(pub, msg []byte, rx, s *big.Int) bool {
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

	sig, err := schnorr.ParseSignature(b[:])
	if err != nil {
		return false
	}

	return sig.Verify(msg, pubKey)
}
