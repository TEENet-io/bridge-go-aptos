package multisig

import (
	"math/big"
)

// SchnorrSigner performs basic sign and can provide pubkey (X, Y) to the viewer.
type SchnorrSigner interface {

	// Sign the message and return the signature in 32byte, 32byte, error
	Sign(message []byte) (*big.Int, *big.Int, error)

	// Get the public key of the signer in X, Y format.
	Pub() (*big.Int, *big.Int, error)
}
