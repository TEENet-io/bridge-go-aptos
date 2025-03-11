package multisig_client

import (
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
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
func (rsw *RemoteSchnorrSigner) Sign(msgHash []byte) (*schnorr.Signature, error) {
	signature, err := rsw.connector.GetSignature(msgHash)
	if err != nil {
		return nil, err
	}
	sig, err := schnorr.ParseSignature(signature)
	if err != nil {
		return nil, err
	}
	return sig, nil
}

// Pub
// Return the public key of the wallet.
func (rsw *RemoteSchnorrSigner) Pub() (*btcec.PublicKey, error) {
	content, err := rsw.connector.GetPubKey()
	if err != nil {
		return nil, err
	}
	if len(content) != 64 {
		return nil, errors.New("invalid content length from server, expected 64 bytes")
	}
	pubkeyBytes := make([]byte, 65)
	pubkeyBytes[0] = 0x04
	copy(pubkeyBytes[1:], content)
	pubkey, err := btcec.ParsePubKey(pubkeyBytes)
	if err != nil {
		return nil, err
	}

	return pubkey, nil
}
