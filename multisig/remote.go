package multisig

import (
	"errors"
	"math/big"

	mgr "github.com/TEENet-io/bridge-go/ethtxmanager"
)

// Define
type RemoteSchnorrWallet struct {
	connector *Connector
}

// Creation
func NewRemoteSchnorrWallet(connector *Connector) *RemoteSchnorrWallet {
	return &RemoteSchnorrWallet{connector: connector}
}

// Sign
func (rsw *RemoteSchnorrWallet) Sign(request *mgr.SignatureRequest, ch chan<- *mgr.SignatureRequest) error {
	sig, err := rsw.connector.GetSignature(request.SigningHash[:])
	if err != nil {
		return err
	}
	rx, s, err := convertSignature(sig)
	if err != nil {
		return err
	}

	ch <- &mgr.SignatureRequest{
		Id:          request.Id,
		SigningHash: request.SigningHash,
		Rx:          rx,
		S:           s,
	}
	return nil
}

// Pub
// Return the public key of the wallet.
// Should be 65 bytes uncompressed [0x04 + x (32byte) + y (32byte)].
func (rsw *RemoteSchnorrWallet) Pub() ([]byte, error) {
	content, err := rsw.connector.GetPubKey()
	if err != nil {
		return nil, err
	}
	if len(content) != 64 {
		return nil, errors.New("invalid content length from server, expected 64 bytes, got " + string(len(content)))
	}
	return append([]byte{0x04}, content...), nil
}

// Helper function.
// Remote signature is of 64 bytes (128 characters in hex)
// We separate the signature into (rx, s) two parts.
func convertSignature(content []byte) (*big.Int, *big.Int, error) {
	if len(content) != 64 {
		return nil, nil, errors.New("invalid content length")
	}
	return new(big.Int).SetBytes(content[:32]), new(big.Int).SetBytes(content[32:]), nil
}
