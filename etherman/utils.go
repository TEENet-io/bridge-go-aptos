package etherman

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// TODO possible duplicate code
// single private key sign version of schnorr signature
// return (rx, s)
func Sign(sk *btcec.PrivateKey, msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(sk, msg[:])
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}
