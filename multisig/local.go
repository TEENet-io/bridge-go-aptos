package multisig

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// Define a local schnorr wallet, which is backed by one single private key.
type LocalSchnorrWallet struct {
	Sk  *btcec.PrivateKey // Private key for schnorr signature (simulation of multi-party)
	PkX *big.Int          // X of the pubkey, used in smart contract creation on Ethereum
	PkY *big.Int          // Y of the pubkey
}

// If user provides a 256-bit (32byte) private key, we can create a schnorr wallet.
func NewLocalSchnorrWallet(privkey []byte) (*LocalSchnorrWallet, error) {
	sk, pk := btcec.PrivKeyFromBytes(privkey)
	return &LocalSchnorrWallet{
		Sk:  sk,
		PkX: pk.X(),
		PkY: pk.Y(),
	}, nil
}

// If user choose to randomly generate a wallet.
func NewRandomLocalSchnorrWallet() (*LocalSchnorrWallet, error) {
	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	return &LocalSchnorrWallet{
		Sk:  sk,
		PkX: sk.PubKey().X(),
		PkY: sk.PubKey().Y(),
	}, nil
}

// Make a schnorr signature.
func (lsw *LocalSchnorrWallet) Sign(message []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(lsw.Sk, message)
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	rx, s := BytesToBigInt(bytes[:32]), BytesToBigInt(bytes[32:])
	return rx, s, nil
}

// Return the (X, Y) of the corresponding public key.
func (lsw *LocalSchnorrWallet) Pub() (*big.Int, *big.Int, error) {
	return lsw.PkX, lsw.PkY, nil
}
