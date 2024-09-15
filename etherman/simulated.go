package etherman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

var (
	simulatedChainID = big.NewInt(1337)
	blockGasLimit    = uint64(999999999999999999)

	btcAddrs = []string{
		"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		"1HLoD9E4SDFFPDiYfNYnkBLQ85Y51J3Zb1",
		"1FvzCLoTPGANNjLgEB5D7e4JZCZ3fK5cP1",
	}
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
	genesisAlloc := map[ethcommon.Address]types.Account{}
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

type ParamConfig struct {
	Deployer  int
	Receiver  int
	Sender    int
	Requester int

	Amount *big.Int
}
type TestEnv struct {
	Sim      *SimulatedChain
	Sk       *btcec.PrivateKey
	Etherman *Etherman
}

func NewTestEnv() *TestEnv {
	sim := NewSimulatedChain()
	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil
	}

	pk := sk.PubKey().X()
	address, _, contract, err := bridge.DeployTEENetBtcBridge(sim.Accounts[0], sim.Backend.Client(), pk)
	if err != nil {
		return nil
	}
	sim.Backend.Commit()

	_pk, err := contract.Pk(nil)
	if err != nil {
		return nil
	}
	if pk.Cmp(_pk) != 0 {
		return nil
	}

	etherman := &Etherman{
		ethClient:     sim.Backend.Client(),
		bridgeAddress: address,
	}

	return &TestEnv{
		Sim:      sim,
		Sk:       sk,
		Etherman: etherman,
	}
}

func (env *TestEnv) GenMintParams(cfg *ParamConfig) *MintParams {
	sim := env.Sim
	sk := env.Sk

	btcTxId := common.RandBytes32()
	if len(btcTxId) == 0 {
		return nil
	}
	receiver := sim.Accounts[cfg.Receiver].From

	msg := crypto.Keccak256Hash(common.EncodePacked(btcTxId, receiver.String(), cfg.Amount)).Bytes()
	rx, s, err := Sign(sk, msg[:])
	if err != nil {
		return nil
	}

	return &MintParams{
		Auth:     sim.Accounts[cfg.Deployer],
		BtcTxId:  btcTxId,
		Amount:   cfg.Amount,
		Receiver: receiver,
		Rx:       rx,
		S:        s,
	}
}

func (env *TestEnv) GenRequestParams(cfg *ParamConfig) *RequestParams {
	return &RequestParams{
		Auth:     env.Sim.Accounts[cfg.Sender],
		Amount:   cfg.Amount,
		Receiver: btcAddrs[0],
	}
}

func (env *TestEnv) GenPrepareParams(cfg *ParamConfig) *PrepareParams {
	txHash := common.RandBytes32()
	if len(txHash) == 0 {
		return nil
	}
	requester := env.Sim.Accounts[cfg.Requester].From
	receiver := btcAddrs[0]
	outpointTxIds := [][32]byte{}
	for i := 0; i < 2; i++ {
		txId := common.RandBytes32()
		if len(txId) == 0 {
			return nil
		}
		outpointTxIds = append(outpointTxIds, txId)
	}
	outpointTxIndices := []*big.Int{big.NewInt(0), big.NewInt(1)}

	msg := crypto.Keccak256Hash(common.EncodePacked(
		txHash, requester.String(), string(receiver), cfg.Amount, outpointTxIds, outpointTxIndices)).Bytes()
	rx, s, err := Sign(env.Sk, msg[:])
	if err != nil {
		return nil
	}

	return &PrepareParams{
		Auth:                env.Sim.Accounts[cfg.Sender],
		RedeemRequestTxHash: txHash,
		Requester:           requester,
		Receiver:            receiver,
		Amount:              cfg.Amount,
		OutpointTxIds:       outpointTxIds,
		OutpointIdxs:        []uint16{0, 1},
		Rx:                  rx,
		S:                   s,
	}
}
