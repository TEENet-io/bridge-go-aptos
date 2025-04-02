// This file contains
// MockedSchnorrThresholdSigner (local version) that implements the SchnorrThresholdSigner interface
// RemoteSchnorrThresholdSigner (remote version) that implements the SchnorrThresholdSigner interface
package signers

import (
	"github.com/TEENet-io/bridge-go/agreement"
	m "github.com/TEENet-io/bridge-go/multisig_client"
)

// Implementation: Local single key schnorr signer
type MockedSchnorrAsyncSigner struct {
	ss m.SchnorrSigner
}

// Create a random mocked local schnorr async signer
func NewRandomMockedSchnorrAsyncSigner() (*MockedSchnorrAsyncSigner, error) {
	ss, err := m.NewRandomLocalSchnorrSigner()
	if err != nil {
		return nil, err
	}
	return &MockedSchnorrAsyncSigner{ss: ss}, nil
}

// Create a mocked local schnorr async signer from a choosen SchnorrSigner
// This signer can be local or remote signer.
func NewMockedSchnorrAsyncSigner(ss m.SchnorrSigner) *MockedSchnorrAsyncSigner {
	return &MockedSchnorrAsyncSigner{ss: ss}
}

// Implementation: Perform Async Signing.
func (mstw *MockedSchnorrAsyncSigner) SignAsync(
	request *agreement.SignatureRequest,
	ch chan<- *agreement.SignatureRequest,
) error {
	_sig, err := mstw.ss.Sign(request.SigningHash[:])

	if err != nil {
		return err
	}

	rx, s, err := m.ConvertSigToRS(_sig)
	if err != nil {
		return err
	}

	ch <- &agreement.SignatureRequest{
		Id:          request.Id,
		SigningHash: request.SigningHash,
		Rx:          rx,
		S:           s,
	}

	return nil
}

// Implementation: Remote schnorr signer
type RemoteSchnorrAsyncSigner struct {
	ss m.SchnorrSigner
}

// Create a remote schnorr threshold signer from a designated SchnorrSigner
func NewRemoteSchnorrAsyncSigner(ss m.SchnorrSigner) *RemoteSchnorrAsyncSigner {
	return &RemoteSchnorrAsyncSigner{ss: ss}
}

// Implementation: Perform Async Signing.
func (rstw *RemoteSchnorrAsyncSigner) SignAsync(
	request *agreement.SignatureRequest,
	ch chan<- *agreement.SignatureRequest,
) error {
	_sig, err := rstw.ss.Sign(request.SigningHash[:])

	if err != nil {
		return err
	}

	rx, s, err := m.ConvertSigToRS(_sig)
	if err != nil {
		return err
	}

	ch <- &agreement.SignatureRequest{
		Id:          request.Id,
		SigningHash: request.SigningHash,
		Rx:          rx,
		S:           s,
	}

	return nil
}
