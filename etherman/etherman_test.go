package etherman

import (
	"context"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/btcsuite/btcd/btcec/v2"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

var btcAddrs = []string{
	"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
	"1HLoD9E4SDFFPDiYfNYnkBLQ85Y51J3Zb1",
	"1FvzCLoTPGANNjLgEB5D7e4JZCZ3fK5cP1",
}

type testEnv struct {
	sim      *SimulatedChain
	sk       *btcec.PrivateKey
	etherman *Etherman
}

type paramConfig struct {
	deployer  int
	receiver  int
	sender    int
	requester int

	amount *big.Int
}

func newTestEnv(t *testing.T) *testEnv {
	sim := NewSimulatedChain()
	sk, err := btcec.NewPrivateKey()
	assert.NoError(t, err)

	pk := sk.PubKey().X()
	address, _, contract, err := bridge.DeployTEENetBtcBridge(sim.Accounts[0], sim.Backend.Client(), pk)
	assert.NoError(t, err)
	sim.Backend.Commit()

	_pk, err := contract.Pk(nil)
	assert.NoError(t, err)
	assert.Equal(t, pk, _pk)

	etherman := &Etherman{
		ethClient:     sim.Backend.Client(),
		bridgeAddress: address,
	}

	return &testEnv{
		sim:      sim,
		sk:       sk,
		etherman: etherman,
	}
}

func TestGetEventMinted(t *testing.T) {
	env := newTestEnv(t)
	sim := env.sim
	etherman := env.etherman

	mintParams := prepareMintParams(t, env, &paramConfig{deployer: 0, receiver: 1, amount: big.NewInt(100)})
	err := etherman.Mint(mintParams)
	assert.NoError(t, err)

	prepareParams := prepparePrepareParams(t, env, &paramConfig{sender: 3, requester: 4, amount: big.NewInt(400)})
	err = etherman.RedeemPrepare(prepareParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	num := curentBlockNum(t, env)

	minted, requested, prepared, err := etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 1)
	assert.Len(t, requested, 0)
	assert.Len(t, prepared, 1)

	checkMintedEvent(t, &minted[0], mintParams)
	checkPreparedEvent(t, &prepared[0], prepareParams)

	err = etherman.TWBTCApprove(sim.Accounts[1], big.NewInt(80))
	assert.NoError(t, err)
	sim.Backend.Commit()

	requestParams := prepareRequestParams(env, &paramConfig{sender: 1, amount: big.NewInt(80)})
	err = etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	num = curentBlockNum(t, env)
	minted, requested, prepared, err = etherman.GetEventLogs(num)
	assert.NoError(t, err)
	assert.Len(t, minted, 0)
	assert.Len(t, requested, 1)
	assert.Len(t, prepared, 0)

	checkRequestedEvent(t, &requested[0], requestParams)
}

func TestRedeemPrepare(t *testing.T) {
	env := newTestEnv(t)
	sim := env.sim
	etherman := env.etherman

	params := prepparePrepareParams(t, env, &paramConfig{sender: 0, requester: 1, amount: big.NewInt(100)})
	err := etherman.RedeemPrepare(params)
	assert.NoError(t, err)
	sim.Backend.Commit()
}

func TestRedeemRequest(t *testing.T) {
	env := newTestEnv(t)
	sim := env.sim
	etherman := env.etherman

	// Mint tokens
	minParams := prepareMintParams(t, env, &paramConfig{deployer: 0, receiver: 1, amount: big.NewInt(100)})
	err := etherman.Mint(minParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	// Approve tokens to bridge
	err = etherman.TWBTCApprove(sim.Accounts[1], big.NewInt(80))
	assert.NoError(t, err)
	sim.Backend.Commit()

	allowance, err := etherman.TWBTCAllowance(sim.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(80), allowance)

	// Request redeem
	requestParams := prepareRequestParams(env, &paramConfig{sender: 1, amount: big.NewInt(80)})
	err = etherman.RedeemRequest(requestParams)
	assert.NoError(t, err)
	sim.Backend.Commit()

	balance, err := etherman.TWBTCBalanceOf(sim.Accounts[1].From)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(20), balance)
}

func TestMint(t *testing.T) {
	env := newTestEnv(t)
	sim := env.sim
	etherman := env.etherman

	params := prepareMintParams(t, env, &paramConfig{deployer: 0, receiver: 1, amount: big.NewInt(100)})
	err := etherman.Mint(params)
	assert.NoError(t, err)
	sim.Backend.Commit()

	balance, err := etherman.TWBTCBalanceOf(ethcommon.HexToAddress(string(params.Receiver)))
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100), balance)
}

func TestGetLatestFinalizedBlockNumber(t *testing.T) {
	etherman, err := NewEtherman(&Config{
		URL:                   "https://mainnet.infura.io/v3/f37af697a9dd4cbfa7e22aaacce33e50",
		BridgeContractAddress: "0x0000000000000000000000000000000000000000",
	})
	assert.NoError(t, err)

	b, err := etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.NotNil(t, b)
	assert.NotZero(t, b)
}

func TestDebugGetLatestFinalizedBlockNumber(t *testing.T) {
	env := newTestEnv(t)
	etherman := env.etherman

	debug = true

	b, err := etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, b, big.NewInt(1))

	env.sim.Backend.Commit()

	b, err = etherman.GetLatestFinalizedBlockNumber()
	assert.NoError(t, err)
	assert.Equal(t, b, big.NewInt(2))

	debug = false
}

func prepareMintParams(t *testing.T, env *testEnv, cfg *paramConfig) *MintParams {
	sim := env.sim
	sk := env.sk

	btcTxIdBytes := make([]byte, 32)
	n, err := rand.Read(btcTxIdBytes)
	assert.NoError(t, err)
	assert.Equal(t, 32, n)

	btcTxId := "0x" + ethcommon.Bytes2Hex(btcTxIdBytes)
	receiver := sim.Accounts[cfg.receiver].From.String()

	msg := crypto.Keccak256Hash(common.EncodePacked(btcTxId, receiver, cfg.amount)).Bytes()
	rxBigInt, sBigInt, err := Sign(sk, msg[:])
	assert.NoError(t, err)
	rx := "0x" + rxBigInt.Text(16)
	s := "0x" + sBigInt.Text(16)

	return &MintParams{
		Auth:     sim.Accounts[cfg.deployer],
		BtcTxId:  Bytes32Hex(btcTxId),
		Amount:   cfg.amount,
		Receiver: AddressHex(receiver),
		Rx:       Bytes32Hex(rx),
		S:        Bytes32Hex(s),
	}
}

func prepareRequestParams(env *testEnv, cfg *paramConfig) *RequestParams {
	return &RequestParams{
		Auth:     env.sim.Accounts[cfg.sender],
		Amount:   cfg.amount,
		Receiver: BTCAddress(btcAddrs[0]),
	}
}

func prepparePrepareParams(t *testing.T, env *testEnv, cfg *paramConfig) *PrepareParams {
	txHash := randBytes32Hex(t)
	requester := env.sim.Accounts[cfg.requester].From.String()
	outpointTxIdsStr := []string{randBytes32Hex(t), randBytes32Hex(t)}
	outpointTxIndices := []*big.Int{big.NewInt(0), big.NewInt(1)}

	msg := crypto.Keccak256Hash(common.EncodePacked(txHash, requester, cfg.amount, outpointTxIdsStr, outpointTxIndices)).Bytes()
	rxBigInt, sBigInt, err := Sign(env.sk, msg[:])
	assert.NoError(t, err)
	rx := "0x" + rxBigInt.Text(16)
	s := "0x" + sBigInt.Text(16)

	var outpointTxIds []Bytes32Hex
	for _, txId := range outpointTxIdsStr {
		outpointTxIds = append(outpointTxIds, Bytes32Hex(txId))
	}

	return &PrepareParams{
		Auth:          env.sim.Accounts[cfg.sender],
		TxHash:        Bytes32Hex(txHash),
		Requester:     AddressHex(requester),
		Amount:        cfg.amount,
		OutpointTxIds: outpointTxIds,
		OutpointIdxs:  []uint16{0, 1},
		Rx:            rx,
		S:             s,
	}
}

func randBytes32Hex(t *testing.T) string {
	b := make([]byte, 32)
	n, err := rand.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 32, n)
	return "0x" + ethcommon.Bytes2Hex(b)
}

func curentBlockNum(t *testing.T, env *testEnv) *big.Int {
	block, err := env.sim.Backend.Client().BlockNumber(context.Background())
	assert.NoError(t, err)
	return big.NewInt(int64(block))
}

func checkMintedEvent(t *testing.T, ev *bridge.TEENetBtcBridgeMinted, params *MintParams) {
	assert.Equal(t, "0x"+ethcommon.Bytes2Hex(ev.BtcTxId[:]), string(params.BtcTxId))
	assert.Equal(t, ev.Receiver.String(), string(params.Receiver))
	assert.Equal(t, ev.Amount, params.Amount)
}

func checkPreparedEvent(t *testing.T, ev *bridge.TEENetBtcBridgeRedeemPrepared, params *PrepareParams) {
	assert.Equal(t, "0x"+ethcommon.Bytes2Hex(ev.EthTxHash[:]), string(params.TxHash))
	assert.Equal(t, ev.Requester.String(), string(params.Requester))
	assert.Equal(t, ev.Amount, params.Amount)
	for i, txId := range ev.OutpointTxIds {
		assert.Equal(t, "0x"+ethcommon.Bytes2Hex(txId[:]), string(params.OutpointTxIds[i]))
	}
	for i, idx := range ev.OutpointIdxs {
		assert.Equal(t, idx, params.OutpointIdxs[i])
	}
}

func checkRequestedEvent(t *testing.T, ev *bridge.TEENetBtcBridgeRedeemRequested, params *RequestParams) {
	assert.Equal(t, ev.Sender.String(), params.Auth.From.String())
	assert.Equal(t, ev.Amount, params.Amount)
	assert.Equal(t, ev.Receiver, string(params.Receiver))
}
