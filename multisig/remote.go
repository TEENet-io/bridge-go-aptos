package multisig

import (
	"errors"
	"math/big"
)

// Define
type RemoteSchnorrSigner struct {
	connector *Connector
}

// Creation via a provided connector
func NewRemoteSchnorrSigner(connector *Connector) *RemoteSchnorrSigner {
	return &RemoteSchnorrSigner{connector: connector}
}

// Sign
func (rsw *RemoteSchnorrSigner) Sign(message []byte) (*big.Int, *big.Int, error) {
	content, err := rsw.connector.GetSignature(message)
	if err != nil {
		return nil, nil, err
	}
	return convertSignature(content)
}

// Pub
// Return the public key of the wallet.
// (X, Y)
func (rsw *RemoteSchnorrSigner) Pub() (*big.Int, *big.Int, error) {
	content, err := rsw.connector.GetPubKey()
	if err != nil {
		return nil, nil, err
	}
	if len(content) != 64 {
		return nil, nil, errors.New("invalid content length from server, expected 64 bytes")
	}
	return BytesToBigInt(content[:32]), BytesToBigInt(content[32:]), nil
}

// Helper function.
// Remote signature is of 64 bytes (128 characters in hex)
// We separate the signature into (rx, s) two parts.
func convertSignature(content []byte) (*big.Int, *big.Int, error) {
	if len(content) != 64 { // currently the server responses with a 64 byte hex string as Schnorr signature.
		return nil, nil, errors.New("invalid content length")
	}
	return BytesToBigInt(content[:32]), BytesToBigInt(content[32:]), nil
}
