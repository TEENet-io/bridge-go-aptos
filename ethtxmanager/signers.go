// This file contains
// 1. SchnorrThresholdSigner interface
// 2. MockedSchnorrThresholdSigner (local version) that implements the interface
// 3. RemoteSchnorrThresholdSigner (remote version) that implements the interface
package ethtxmanager

import (
	m "github.com/TEENet-io/bridge-go/multisig_client"
)

// This interface is used by eth tx manager
// to interact with the schnorr signer for signing.
// It uses a channel to perform async signing operation.
// It uses an underlying ACTUAL signer to do the job.
type SchnorrAsyncSigner interface {
	// Sign sends a request to the signer to sign on the signing hash
	// and return the signature via the provided channel
	SignAsync(request *SignatureRequest, ch chan<- *SignatureRequest) error
}

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
	request *SignatureRequest,
	ch chan<- *SignatureRequest,
) error {
	_sig, err := mstw.ss.Sign(request.SigningHash[:])

	if err != nil {
		return err
	}

	rx, s, err := m.ConvertSigToRS(_sig)
	if err != nil {
		return err
	}

	ch <- &SignatureRequest{
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
	request *SignatureRequest,
	ch chan<- *SignatureRequest,
) error {
	_sig, err := rstw.ss.Sign(request.SigningHash[:])

	if err != nil {
		return err
	}

	rx, s, err := m.ConvertSigToRS(_sig)
	if err != nil {
		return err
	}

	ch <- &SignatureRequest{
		Id:          request.Id,
		SigningHash: request.SigningHash,
		Rx:          rx,
		S:           s,
	}

	return nil
}
