package etherman

import (
	"context"
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
		"bc1qngcgpqcxfc0pq0dhkq3qknwqjte0yrawharxjm",
		"14uFymMQ43y9TvCY5ZJC2dAB9n16cErfUz",
		"bc1q7fpy8z8cpmx7qzvwac7lrp0vacqflnh4xpa9nx",
		"bc1qee3s2t2kt5qmgddlt6wdh2dmlh9el9feazrzda",
		"bc1q28pgr603dspc3ap88gdkzx25dl64ygz8p4y40p",
		"31wn4XQAJLxyRCKs21hVqsio2iAJNLELQc",
		"1NDxDDSHVHvSv48vd27NNHkXHYZjDdVLss",
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
	// index of the accounts in the simulated chain
	// 		< 0 	== accounts[0]
	// 		[0, 9] 	== accounts[i]
	// 		> 9		== accounts[9]
	Receiver  int
	Requester int

	Amount *big.Int

	// Number of randomly generated outpoints
	OutpointNum int

	// Index of the 10 BTC addresses stored for testing
	// 		< 0 	== random and invalid
	// 		[0, 9] 	== btcAddrs[i]
	// 		> 0		== btcAddrs[9]
	BtcAddrIdx int
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
	idx := cfg.Receiver
	if idx < 0 {
		idx = 0
	}
	if idx > 9 {
		idx = 9
	}
	receiver := chain.Accounts[idx].From

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
	idx1 := cfg.Requester
	if idx1 < 0 {
		idx1 = 0
	}
	if idx1 > 9 {
		idx1 = 9
	}

	idx2 := cfg.BtcAddrIdx
	if idx2 > 9 {
		idx2 = 9
	}

	if cfg.BtcAddrIdx < 0 {
		return &RequestParams{
			Amount:   cfg.Amount,
			Receiver: "invalid_btc_address",
		}
	} else {
		return &RequestParams{
			Amount:   cfg.Amount,
			Receiver: btcAddrs[idx2],
		}
	}
}

func (env *SimEtherman) GenPrepareParams(cfg *ParamConfig) (p *PrepareParams) {
	idx := cfg.BtcAddrIdx
	if idx < 0 {
		idx = 0
	}
	if idx > 9 {
		idx = 9
	}
	receiver := btcAddrs[idx]

	idx = cfg.Requester
	if idx < 0 {
		idx = 0
	}
	if idx > 9 {
		idx = 9
	}
	requester := env.Chain.Accounts[idx].From

	txHash := common.RandBytes32()
	outpointTxIds := []ethcommon.Hash{}
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

func (env *SimEtherman) Mint(receiver int, amount int) (ethcommon.Hash, *MintParams) {
	params := env.GenMintParams(&ParamConfig{
		Receiver: receiver,
		Amount:   big.NewInt(int64(amount)),
	})
	tx, err := env.Etherman.Mint(params)
	if err != nil {
		panic(err)
	}

	return tx.Hash(), params
}

func (env *SimEtherman) Approve(requester int, amount int) ethcommon.Hash {
	balBefore, err := env.Etherman.TWBTCBalanceOf(env.Chain.Accounts[requester].From)
	if err != nil {
		panic(err)
	}
	if balBefore.Uint64() < uint64(amount) {
		panic("insufficient balance")
	}

	txHash, err := env.Etherman.TWBTCApprove(env.Chain.Accounts[requester], big.NewInt(int64(amount)))
	if err != nil {
		panic(err)
	}

	return txHash
}

func (env *SimEtherman) Request(auth *bind.TransactOpts, requester int, amount int, btcAddrIdx int) (ethcommon.Hash, *RequestParams) {
	allowed, err := env.Etherman.TWBTCAllowance(env.Chain.Accounts[requester].From)
	if err != nil {
		panic(err)
	}
	if allowed.Uint64() < uint64(amount) {
		panic("insufficient allowance")
	}

	params := env.GenRequestParams(&ParamConfig{
		Requester:  requester,
		Amount:     big.NewInt(int64(amount)),
		BtcAddrIdx: btcAddrIdx,
	})
	tx, err := env.Etherman.RedeemRequest(auth, params)
	if err != nil {
		panic(err)
	}

	return tx.Hash(), params
}

func (env *SimEtherman) Prepare(
	requester int, amount int, btcAddrIdx int, outpointNum int,
) (ethcommon.Hash, *PrepareParams) {
	params := env.GenPrepareParams(&ParamConfig{
		Requester:   requester,
		Amount:      big.NewInt(int64(amount)),
		BtcAddrIdx:  btcAddrIdx,
		OutpointNum: outpointNum,
	})
	tx, err := env.Etherman.RedeemPrepare(params)
	if err != nil {
		panic(err)
	}

	return tx.Hash(), params
}

func (env *SimEtherman) GetAuth(idx int) *bind.TransactOpts {
	if idx < 0 || idx > 9 {
		panic("invalid account index")
	}

	auth := env.Chain.Accounts[idx]
	nonce, err := env.Chain.Backend.Client().PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		panic(err)
	}

	auth.Nonce = big.NewInt(int64(nonce))

	return auth
}

func (env *SimEtherman) UpdateBackendAccountNonce() {
	nonce, err := env.Chain.Backend.Client().PendingNonceAt(context.Background(), env.Chain.Accounts[0].From)
	if err != nil {
		panic(err)
	}
	env.Etherman.SetNonce(nonce)
}
