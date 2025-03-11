package multisig_client

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// Define a local schnorr wallet, which is backed by one single private key.
type LocalSchnorrSigner struct {
	Sk *btcec.PrivateKey // Private key for schnorr signature (simulation of multi-party)
	Pk *btcec.PublicKey  // Public key for schnorr signature (simulation of multi-party)
}

// If user provides a 256-bit (32byte) private key, we can create a schnorr wallet.
func NewLocalSchnorrSigner(privkey []byte) (*LocalSchnorrSigner, error) {
	sk, pk := btcec.PrivKeyFromBytes(privkey)
	return &LocalSchnorrSigner{
		Sk: sk, Pk: pk,
	}, nil
}

// If user choose to randomly generate a wallet.
func NewRandomLocalSchnorrSigner() (*LocalSchnorrSigner, error) {
	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	return &LocalSchnorrSigner{
		Sk: sk, Pk: sk.PubKey(),
	}, nil
}

// Make a schnorr signature.
func (lsw *LocalSchnorrSigner) Sign(msgHash []byte) (*schnorr.Signature, error) {
	sig, err := schnorr.Sign(lsw.Sk, msgHash)
	if err != nil {
		return nil, err
	} else {
		return sig, nil
	}
}

// Return the (X, Y) of the corresponding public key.
func (lsw *LocalSchnorrSigner) Pub() (*btcec.PublicKey, error) {
	return lsw.Pk, nil
}
