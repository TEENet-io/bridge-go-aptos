package multisig_client

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// SchnorrSigner performs basic sign and can provide pubkey (X, Y) to the viewer.
type SchnorrSigner interface {

	// Sign the message and return the signature, error
	Sign(msgHash []byte) (*schnorr.Signature, error)

	// Get the public key of the signer in public key format.
	Pub() (*btcec.PublicKey, error)
}
