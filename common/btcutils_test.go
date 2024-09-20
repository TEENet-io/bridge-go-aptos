package common

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
)

func TestSchnorrSignature(t *testing.T) {
	hash := RandBytes32()
	privKey, _ := btcec.NewPrivateKey()
	rx, s, err := Sign(privKey, hash[:])
	assert.NoError(t, err)

	pubKey := privKey.PubKey().X().Bytes()
	ok := Verify(pubKey, hash[:], rx, s)
	assert.True(t, ok)
}
