package multisig_client

import (
	"crypto/rand"
	"testing"
)

func TestLocalSignAndVerify(t *testing.T) {
	lss, err := NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("Error creating random local schnorr signer: %v", err)
	}

	msgHash := make([]byte, 32)
	_, err = rand.Read(msgHash)
	if err != nil {
		t.Fatalf("Error generating random message hash: %v", err)
	}
	sig, err := lss.Sign(msgHash)
	if err != nil {
		t.Fatalf("Error signing: %v", err)
	}

	pubKey, err := lss.Pub()
	if err != nil {
		t.Fatalf("Error getting public key: %v", err)
	}

	t.Logf("Public Key: %v\n", pubKey)

	ok := sig.Verify(msgHash, pubKey)
	if !ok {
		t.Fatalf("Signature verification failed")
	}
}
