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
	logger "github.com/sirupsen/logrus"
)

const (
	NUMBER_OF_ACCOUNTS = 10   // Use 10 BTC ccounts for testing
	CHAIN_ID_INT64     = 1337 // Use 1337 as simulated chain id
)

var (
	SimulatedChainID = big.NewInt(CHAIN_ID_INT64)
	blockGasLimit    = uint64(999999999999999999)

	// 10 BTC addresses (to simulate receiver of redeem)
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

// Simulated Chain with execution backend
// and some genesis accounts.
type SimulatedChain struct {
	Backend  *simulated.Backend
	Accounts []*bind.TransactOpts
}

func NewSimulatedChain() *SimulatedChain {
	// create genesis accounts
	nAccount := NUMBER_OF_ACCOUNTS
	accounts := make([]*bind.TransactOpts, nAccount)
	for i := 0; i < nAccount; i++ {
		accounts[i] = newAuth()
	}

	// allocate funds to genesis accounts
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

// Create a new auth object with a random private key.
// The auth object is used to sign transactions
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
	Receiver  int // ethereum account index (mint receiver)
	Requester int // ethereum account index (redeem requester)

	Amount *big.Int

	// Number of randomly generated outpoints
	OutpointNum int

	// Index of the 10 BTC addresses stored for testing
	// 		< 0 	== random and invalid
	// 		[0, 9] 	== btcAddrs[i]
	// 		> 0		== btcAddrs[9]
	BtcAddrIdx int // btc address index
}

type SimEtherman struct {
	Chain    *SimulatedChain
	Sk       *btcec.PrivateKey // Private key for schnorr signature (simulation of multi-party)
	Etherman *Etherman
}

func NewSimEtherman() (*SimEtherman, error) {
	chain := NewSimulatedChain()

	// Random bitcoin private key.
	// TODO: Change to a multi-party schnorr private key.
	sk, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	// X of the pubkey (this is actually a simulation of a multi-party schnorr pubkey)
	// TODO: Change to a multi-party schnorr pubkey aggregation.
	pk := sk.PubKey().X()

	// Deploy the bridge contract.
	// Pubkey is embedded in the bridge contract.
	// Later the smart contract use this pubkey to verify the validity of signature (Rx, s) of every request.
	bridgeAddress, _, contract, err := bridge.DeployTEENetBtcBridge(chain.Accounts[0], chain.Backend.Client(), pk)
	if err != nil {
		return nil, err
	}
	// move the chain to the next block
	chain.Backend.Commit()

	// bridgeContract is a wrapper around the deployed bridge contract
	bridgeContract, err := bridge.NewTEENetBtcBridge(bridgeAddress, chain.Backend.Client())
	if err != nil {
		return nil, err
	}

	// TWBTC contract address
	twbtcAddress, err := bridgeContract.Twbtc(nil)
	if err != nil {
		return nil, err
	}

	// Compare bridge contract public key with the one we provided
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

// Generate Mint parameters from ParamConfig.
// Translate cfg.Receiver (idx) to a pre-stored ethereum receiver address
func (env *SimEtherman) GenMintParams(cfg *ParamConfig, btcTxId [32]byte) *MintParams {
	chain := env.Chain
	sk := env.Sk

	idx := cfg.Receiver
	if idx < 0 {
		idx = 0
	}
	if idx > 9 {
		idx = 9
	}
	receiver := chain.Accounts[idx].From

	// Create (rx, s) schnorr signature of (btctxid, ethaddr, amount)
	content := crypto.Keccak256Hash(common.EncodePacked(btcTxId, receiver.String(), cfg.Amount)).Bytes()
	rx, s, err := Sign(sk, content[:])
	if err != nil {
		return nil
	}

	// Assemble Mint parameters
	return &MintParams{
		BtcTxId:  btcTxId,
		Amount:   cfg.Amount,
		Receiver: receiver,
		Rx:       rx,
		S:        s,
	}
}

// Generate a Request (= RedeemRequest) parameters from ParamConfig.
// cfg.Requester = idx of the ethererm requester account in the simulated chain.
// cfg.BtcAddrIdx = idx of the BTC address in the btcAddrs array or 'invalid_btc_address'.
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

// Generatea Prepare parameters from ParamConfig.
// cfg.BtcAddrIdx = idx of the BTC address in the btcAddrs array.
// cfg.Requester = idx of the ethereum requester account in the simulated chain.
// !!! It fakes random requestTxHash, outpointTxIds, outpointIdxs.
func (env *SimEtherman) GenPrepareParams(cfg *ParamConfig) (p *PrepareParams) {
	idx := cfg.BtcAddrIdx
	if idx < 0 {
		idx = 0
	}
	if idx > 9 {
		idx = 9
	}
	receiver := btcAddrs[idx]

	idx2 := cfg.Requester
	if idx2 < 0 {
		idx2 = 0
	}
	if idx2 > 9 {
		idx2 = 9
	}
	requester := env.Chain.Accounts[idx2].From

	// TODO: Use real ETH Request Transaction Hash
	reqTxHash := common.RandBytes32()
	outpointTxIds := []ethcommon.Hash{}
	outpointIdxs := []uint16{}

	for i := 0; i < cfg.OutpointNum; i++ {
		// TODO: Use real BTC Outpoint Transaction Hash
		outpointTxIds = append(outpointTxIds, common.RandBytes32())
		// TODO: Use real BTC Outpoint VOUT
		outpointIdxs = append(outpointIdxs, uint16(i))
	}

	p = &PrepareParams{
		RequestTxHash: reqTxHash,
		Requester:     requester,
		Receiver:      receiver,
		Amount:        cfg.Amount,
		OutpointTxIds: outpointTxIds,
		OutpointIdxs:  outpointIdxs,
	}

	// create the hash
	msg := p.SigningHash()
	// sign the hash
	rx, s, err := Sign(env.Sk, msg[:])
	if err != nil {
		return nil
	}

	p.Rx = rx
	p.S = s

	return
}

// single-private-key schnorr signature.
// content is usually the [hash of a message].
// return (rx, s)
func (env *SimEtherman) Sign(content []byte) (*big.Int, *big.Int, error) {
	// Generate a schnorr signature
	// with a signle private key = (rx, s)
	// the signature can be combined with other signatures in real production.
	// Now is only a simulation. So only a single schnorr signature.
	sig, err := schnorr.Sign(env.Sk, content)
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}

// !!! This is a convenient function.
// It selects a pre-stored ethereum account in as the mint receiver.
// and mint some TWBTC to the receiver.
// btcTxId is the bitcoin transaction id that did the deposit on btc side.
func (env *SimEtherman) Mint(btcTxId [32]byte, receiverIdx int, amount int) (ethcommon.Hash, *MintParams) {
	// Attention:
	// Translate from ethereum idx to actual addresses.
	params := env.GenMintParams(
		&ParamConfig{
			Receiver: receiverIdx,
			Amount:   big.NewInt(int64(amount)),
		},
		btcTxId,
	)

	// Call the real mint() on chain.
	tx, err := env.Etherman.Mint(params)
	if err != nil {
		logger.Fatal(err)
	}

	return tx.Hash(), params
}

// !!! This is a convenient function.
// It sends out an Approve() action on Ethereum (user action).
// It selects a pre-stored eth account in as the requester.
// and approves some TWBTC to be spent by our bridge.
func (env *SimEtherman) Approve(requesterIdx int, amount int) ethcommon.Hash {
	// TODO: auth can be passed in? (align with requesterIdx)
	// Otherwise nonce maybe reused and creates conflict.
	auth := env.Chain.Accounts[requesterIdx]

	balBefore, err := env.Etherman.TWBTCBalanceOf(auth.From)
	if err != nil {
		logger.Fatal(err)
	}
	if balBefore.Uint64() < uint64(amount) {
		logger.Fatal("insufficient balance")
	}

	tx, err := env.Etherman.TWBTCApprove(auth, big.NewInt(int64(amount)))
	if err != nil {
		logger.Fatal(err)
	}

	return tx.Hash()
}

// !!! This is a convenient function.
// It sends out a request of redeem on Ethereum (user action).
// Eth Tx sender is auth, shall be align with requesterIdx.
// It selects a pre-stored eth account as the requester.
// It selects a pre-stored btc address as teh receiver.
func (env *SimEtherman) Request(auth *bind.TransactOpts, requesterIdx int, amount int, btcAddrIdx int) (ethcommon.Hash, *RequestParams) {
	// Check the allowance of TWBTC
	allowed, err := env.Etherman.TWBTCAllowance(env.Chain.Accounts[requesterIdx].From)
	if err != nil {
		logger.Fatal(err)
	}
	if allowed.Uint64() < uint64(amount) {
		logger.Fatal("insufficient allowance")
	}

	// Attention:
	// Translate from ethereum idx to actual addresses.
	// Translate from bitcoin idx to actual addresses.
	params := env.GenRequestParams(&ParamConfig{
		Requester:  requesterIdx,
		Amount:     big.NewInt(int64(amount)),
		BtcAddrIdx: btcAddrIdx,
	})
	tx, err := env.Etherman.RedeemRequest(auth, params)
	if err != nil {
		logger.Fatal(err)
	}

	return tx.Hash(), params
}

// Craft a request for redeem on Ethereum.
func (env *SimEtherman) Request2(auth *bind.TransactOpts, requesterIdx int, amount int, btcAddr string) (ethcommon.Hash, *RequestParams) {
	// Check the allowance of TWBTC
	allowed, err := env.Etherman.TWBTCAllowance(env.Chain.Accounts[requesterIdx].From)
	if err != nil {
		logger.Fatal(err)
	}
	if allowed.Uint64() < uint64(amount) {
		logger.Fatal("insufficient allowance")
	}

	// Attention:
	// Translate from ethereum idx to actual addresses.
	// Translate from bitcoin idx to actual addresses.
	params := env.GenRequestParams(&ParamConfig{
		Requester:  requesterIdx,
		Amount:     big.NewInt(int64(amount)),
		BtcAddrIdx: 0,
	})

	// Set the btc receiver address to a real one.
	params.Receiver = btcAddr

	// logger.Debugf("request params: %+v", params)

	tx, err := env.Etherman.RedeemRequest(auth, params)
	if err != nil {
		logger.Fatal(err)
	}

	return tx.Hash(), params
}

// !!! This is a convenient function.
// It sends out a prepare of redeem on Ethereum (bridge action).
// It selects a pre-stored eth account as the requester.
// It selects a pre-stored btc address as teh receiver.
// It fakes the outpointTxIds and outpointIdxs (on btc).
// It also fakes requestTxHash (on ethereum)
func (env *SimEtherman) Prepare(
	requesterIdx int, amount int, btcAddrIdx int, outpointNum int,
) (ethcommon.Hash, *PrepareParams) {
	// Attention:
	// Create fake "redeem prepare" parameters.
	params := env.GenPrepareParams(&ParamConfig{
		Requester:   requesterIdx,
		Amount:      big.NewInt(int64(amount)),
		BtcAddrIdx:  btcAddrIdx,
		OutpointNum: outpointNum,
	})
	// Do the real redeem prepare on chain.
	tx, err := env.Etherman.RedeemPrepare(params)
	if err != nil {
		logger.Fatal(err)
	}

	return tx.Hash(), params
}

// Fetch a pre-stored ethereum account in the simulated chain.
// Set its nonce +1. Very useful if you send multiple tx in a block.
// The nonce won't conflict with each other.
func (env *SimEtherman) GetAuth(idx int) *bind.TransactOpts {
	if idx < 0 || idx > 9 {
		logger.Fatal("invalid account index")
	}

	auth := env.Chain.Accounts[idx]
	nonce, err := env.Chain.Backend.Client().PendingNonceAt(context.Background(), auth.From)
	if err != nil {
		logger.Fatal(err)
	}

	auth.Nonce = big.NewInt(int64(nonce))

	return auth
}
