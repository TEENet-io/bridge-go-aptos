// This file contains
// 1. SchnorrThresholdWallet interface
// 2. MockedSchnorrThresholdWallet (local version) that implements the interface
// 3. RemoteSchnorrThresholdWallet (remote version) that implements the interface
package ethtxmanager

import (
	m "github.com/TEENet-io/bridge-go/multisig_client"
)

// This interface is used by eth tx manager
// to interact with the schnorr wallet for signing.
// It uses a channel to perform async signing operation.
// It uses an underlying ACTUAL signer to do the job.
type SchnorrAsyncWallet interface {
	// Sign sends a request to the wallet to sign on the signing hash
	// and return the signature via the provided channel
	SignAsync(request *SignatureRequest, ch chan<- *SignatureRequest) error
}

// Implementation: Local single key schnorr wallet
type MockedSchnorrAsyncWallet struct {
	ss m.SchnorrSigner
}

// Create a random mocked local schnorr async wallet
func NewRandomMockedSchnorrAsyncWallet() (*MockedSchnorrAsyncWallet, error) {
	ss, err := m.NewRandomLocalSchnorrSigner()
	if err != nil {
		return nil, err
	}
	return &MockedSchnorrAsyncWallet{ss: ss}, nil
}

// Create a mocked local schnorr async wallet from a choosen SchnorrSigner
// This signer can be local or remote signer.
func NewMockedSchnorrAsyncWallet(ss m.SchnorrSigner) *MockedSchnorrAsyncWallet {
	return &MockedSchnorrAsyncWallet{ss: ss}
}

// Implementation: Perform Async Signing.
func (mstw *MockedSchnorrAsyncWallet) SignAsync(
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

// Implementation: Remote schnorr wallet
type RemoteSchnorrAsyncWallet struct {
	ss m.SchnorrSigner
}

// Create a remote schnorr threshold wallet from a designated SchnorrSigner
func NewRemoteSchnorrAsyncWallet(ss m.SchnorrSigner) *RemoteSchnorrAsyncWallet {
	return &RemoteSchnorrAsyncWallet{ss: ss}
}

// Implementation: Perform Async Signing.
func (rstw *RemoteSchnorrAsyncWallet) SignAsync(
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
