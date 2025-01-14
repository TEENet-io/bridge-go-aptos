// This file contains
// 1. SchnorrThresholdWallet interface
// 2. MockedSchnorrThresholdWallet (local version) that implements the interface
// 3. RemoteSchnorrThresholdWallet (remote version) that implements the interface
package ethtxmanager

import (
	m "github.com/TEENet-io/bridge-go/multisig"
)

// This interface is used by eth tx manager
// to interact with the schnorr wallet for signing
// then reply in the channel with result.
// It can use an underlying ACTUAL signer to do the job.
type SchnorrThresholdWallet interface {
	// Sign sends a request to the wallet to sign on the signing hash
	// and return the signature via the provided channel
	SignAsync(request *SignatureRequest, ch chan<- *SignatureRequest) error
}

// Implementation: Local single key schnorr wallet
type MockedSchnorrThresholdWallet struct {
	ss m.SchnorrSigner
}

// Create a random mocked local schnorr threshold wallet
func NewRandomMockedSchnorrThresholdWallet() (*MockedSchnorrThresholdWallet, error) {
	ss, err := m.NewRandomLocalSchnorrSigner()
	if err != nil {
		return nil, err
	}
	return &MockedSchnorrThresholdWallet{ss: ss}, nil
}

// Create a mocked local schnorr threshold wallet from a designated SchnorrSigner
func NewMockedSchnorrThresholdWallet(ss m.SchnorrSigner) *MockedSchnorrThresholdWallet {
	return &MockedSchnorrThresholdWallet{ss: ss}
}

// Implementation. Perform Async Signing.
func (mstw *MockedSchnorrThresholdWallet) SignAsync(
	request *SignatureRequest,
	ch chan<- *SignatureRequest,
) error {
	rx, s, err := mstw.ss.Sign(request.SigningHash[:])

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
type RemoteSchnorrThresholdWallet struct {
	ss m.SchnorrSigner
}

// Create a remote schnorr threshold wallet from a designated SchnorrSigner
func NewRemoteSchnorrThresholdWallet(ss m.SchnorrSigner) *RemoteSchnorrThresholdWallet {
	return &RemoteSchnorrThresholdWallet{ss: ss}
}

// Implementation: Perform Async Signing.
func (rstw *RemoteSchnorrThresholdWallet) SignAsync(
	request *SignatureRequest,
	ch chan<- *SignatureRequest,
) error {
	rx, s, err := rstw.ss.Sign(request.SigningHash[:])

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
