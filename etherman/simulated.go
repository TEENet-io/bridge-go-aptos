package etherman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
)

var (
	SimulatedChainID = big.NewInt(1337)
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
	auth, _ := bind.NewKeyedTransactorWithChainID(sk, SimulatedChainID)
	return auth
}

type ParamConfig struct {
	Receiver  int
	Sender    int
	Requester int

	Amount *big.Int

	OutpointNum int
}
type SimEtherman struct {
	Chain    *SimulatedChain
	Sk       *btcec.PrivateKey
	Etherman *Etherman
}

func NewSimEtherman() (*SimEtherman, error) {
	chain := NewSimulatedChain()
	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	pk := sk.PubKey().X()
	bridgeAddress, _, contract, err := bridge.DeployTEENetBtcBridge(chain.Accounts[0], chain.Backend.Client(), pk)
	if err != nil {
		return nil, err
	}
	chain.Backend.Commit()

	bridgeContract, err := bridge.NewTEENetBtcBridge(bridgeAddress, chain.Backend.Client())
	if err != nil {
		return nil, err
	}
	twbtcAddress, err := bridgeContract.Twbtc(nil)
	if err != nil {
		return nil, err
	}

	_pk, err := contract.Pk(nil)
	if err != nil {
		return nil, err
	}
	if pk.Cmp(_pk) != 0 {
		return nil, err
	}

	cfg := &Config{
		BridgeContractAddress: bridgeAddress,
		TWBTCContractAddress:  twbtcAddress,
	}

	etherman := &Etherman{
		ethClient: chain.Backend.Client(),
		cfg:       cfg,
		auth:      chain.Accounts[0],
	}

	return &SimEtherman{
		Etherman: etherman,
		Sk:       sk,
		Chain:    chain,
	}, nil
}

func (env *SimEtherman) GenMintParams(cfg *ParamConfig) *MintParams {
	chain := env.Chain
	sk := env.Sk

	btcTxId := common.RandBytes32()
	if len(btcTxId) == 0 {
		return nil
	}
	receiver := chain.Accounts[cfg.Receiver].From

	msg := crypto.Keccak256Hash(common.EncodePacked(btcTxId, receiver.String(), cfg.Amount)).Bytes()
	rx, s, err := Sign(sk, msg[:])
	if err != nil {
		return nil
	}

	return &MintParams{
		BtcTxId:  btcTxId,
		Amount:   cfg.Amount,
		Receiver: receiver,
		Rx:       rx,
		S:        s,
	}
}

func (env *SimEtherman) GenRequestParams(cfg *ParamConfig) *RequestParams {
	return &RequestParams{
		Auth:     env.Chain.Accounts[cfg.Sender],
		Amount:   cfg.Amount,
		Receiver: btcAddrs[0],
	}
}

func (env *SimEtherman) GenPrepareParams(cfg *ParamConfig) (p *PrepareParams) {
	txHash := common.RandBytes32()
	requester := env.Chain.Accounts[cfg.Requester].From
	receiver := btcAddrs[0]
	outpointTxIds := [][32]byte{}
	outpointIdxs := []uint16{}

	for i := 0; i < cfg.OutpointNum; i++ {
		outpointTxIds = append(outpointTxIds, common.RandBytes32())
		outpointIdxs = append(outpointIdxs, uint16(i))
	}

	p = &PrepareParams{
		RequestTxHash: txHash,
		Requester:     requester,
		Receiver:      receiver,
		Amount:        cfg.Amount,
		OutpointTxIds: outpointTxIds,
		OutpointIdxs:  outpointIdxs,
	}

	msg := p.SigningHash()
	rx, s, err := Sign(env.Sk, msg[:])
	if err != nil {
		return nil
	}

	p.Rx = rx
	p.S = s

	return
}

func (env *SimEtherman) Sign(msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(env.Sk, msg)
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}
