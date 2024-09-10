package etherman

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

var (
	simulatedChainID = big.NewInt(1337)
	blockGasLimit    = uint64(999999999999999999)
)

type SimulatedChain struct {
	Backend  *simulated.Backend
	Accounts []*bind.TransactOpts
}

func NewSimulatedChain() *SimulatedChain {
	// create accounts
	nAccount := 10
	accounts := make([]*bind.TransactOpts, nAccount)
	for i := 0; i < nAccount; i++ {
		accounts[i] = newAuth()
	}

	// allocate funds to accounts
	genesisAlloc := map[common.Address]types.Account{}
	for _, account := range accounts {
		balance, _ := new(big.Int).SetString("100000000000000000000", 10)
		genesisAlloc[account.From] = types.Account{
			Balance: balance,
		}
	}

	// create simulated backend
	backend := simulated.NewBackend(genesisAlloc, simulated.WithBlockGasLimit(blockGasLimit))

	return &SimulatedChain{
		Backend:  backend,
		Accounts: accounts,
	}
}

func newAuth() *bind.TransactOpts {
	sk, _ := crypto.GenerateKey()
	auth, _ := bind.NewKeyedTransactorWithChainID(sk, simulatedChainID)
	return auth
}
