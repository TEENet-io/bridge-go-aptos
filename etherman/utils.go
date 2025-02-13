package etherman

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
)

// Convert a private key string to an ETH private key object
func StringToPrivateKey(s string) (*ecdsa.PrivateKey, error) {
	sk, err := crypto.HexToECDSA(s)
	return sk, err
}

// Generate one random ETH private key
func GenPrivateKey() *ecdsa.PrivateKey {
	sk, _ := crypto.GenerateKey()
	return sk
}

// Generate several random private keys
func GenPrivateKeys(number int) []*ecdsa.PrivateKey {
	privateKeys := make([]*ecdsa.PrivateKey, number)
	for i := 0; i < number; i++ {
		privateKeys[i] = GenPrivateKey()
	}
	return privateKeys
}

// Create a new auth object with a given private key.
// The auth object is used to sign transactions
func NewAuth(sk *ecdsa.PrivateKey, chainID *big.Int) *bind.TransactOpts {
	auth, _ := bind.NewKeyedTransactorWithChainID(sk, chainID)
	return auth
}

// From eth private keys to eth sign-able accounts
func NewAuthAccounts(privateKeys []*ecdsa.PrivateKey, chainId *big.Int) []*bind.TransactOpts {
	// create genesis accounts
	nAccount := len(privateKeys)
	accounts := make([]*bind.TransactOpts, nAccount)
	for i := 0; i < nAccount; i++ {
		accounts[i] = NewAuth(privateKeys[i], chainId)
	}
	return accounts
}
