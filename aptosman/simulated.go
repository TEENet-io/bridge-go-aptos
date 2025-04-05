package aptosman

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/aptos-labs/aptos-go-sdk"
	"golang.org/x/crypto/ed25519"
)

const (
	TESTNET = "testnet"
	DEVNET  = "devnet"
)

// Create Aptos client
func createSimAptosClient() (*aptos.Client, error) {
	// Get network configuration from environment variable, default to devnet
	networkConfig := aptos.DevnetConfig // TODO mainnet

	// Create client
	client, err := aptos.NewClient(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}
	return client, nil
}

var (
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

type ParamConfig struct {
	// index of the accounts in the simulated chain
	// 		< 0 	== accounts[0]
	// 		[0, 9] 	== accounts[i]
	// 		> 9		== accounts[9]
	Receiver  int // aptos account index (mint receiver)
	Requester int // aptos account index (redeem requester)

	Amount *big.Int

	// Number of randomly generated outpoints
	OutpointNum int

	// Index of the 10 BTC addresses stored for testing
	// 		< 0 	== random and invalid
	// 		[0, 9] 	== btcAddrs[i]
	// 		> 0		== btcAddrs[9]
	BtcAddrIdx int // btc address index
}

type SimAptosman struct {
	Aptosman    *Aptosman
	MultiSigner multisig_client.SchnorrSigner
	Accounts    []*aptos.Account
}

// 创建一个新的模拟 Aptosman 环境
func NewSimAptosman_from_privateKey(privateKey string) (*SimAptosman, error) {
	// 创建管理员账户
	adminAccount, err := createAccountFromPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin account: %v", err)
	}

	// 创建 10 个测试账户
	accounts := []*aptos.Account{adminAccount}
	for i := 0; i < 9; i++ {
		randomBytes := make([]byte, 32)
		_, err := rand.Read(randomBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random bytes: %v", err)
		}
		privateKey := ed25519.NewKeyFromSeed(randomBytes)
		account, err := NewAccount(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create test account %d: %v", i, err)
		}
		accounts = append(accounts, account)
	}

	// 创建 SchnorrSigner
	schnorrSigner, err := multisig_client.NewRandomLocalSchnorrSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create schnorr signer: %v", err)
	}

	// 创建 Aptosman 配置
	cfg := &AptosmanConfig{
		Network:       DEVNET,
		ModuleAddress: "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864",
		URL:           "https://fullnode.testnet.aptoslabs.com",
	}

	// 创建 Aptosman 实例
	aptosman, err := NewAptosman(cfg, adminAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to create Aptosman: %v", err)
	}

	return &SimAptosman{
		Aptosman:    aptosman,
		MultiSigner: schnorrSigner,
		Accounts:    accounts,
	}, nil
}

// // 生成 Mint 参数
// func (env *SimAptosman) GenMintParams(cfg *ParamConfig, btcTxId [32]byte) *MintParams {
// 	idx := cfg.Receiver
// 	if idx < 0 {
// 		idx = 0
// 	}
// 	if idx > 9 {
// 		idx = 9
// 	}
// 	receiver := env.Accounts[idx].Address().String()

// 	// 创建签名内容
// 	content := common.Keccak256Hash(common.EncodePacked(btcTxId, receiver, cfg.Amount)).Bytes()
// 	sig, err := env.MultiSigner.Sign(content[:])
// 	if err != nil {
// 		return nil
// 	}
// 	rx, s, err := multisig_client.ConvertSigToRS(sig)
// 	if err != nil {
// 		return nil
// 	}

// 	return &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   cfg.Amount,
// 		Receiver: receiver,
// 		Rx:       rx,
// 		S:        s,
// 	}
// }

// func (env *SimAptosman) Mint(btcTxId [32]byte, receiverIdx int, amount int) (string, *MintParams) {
// 	params := env.GenMintParams(
// 		&ParamConfig{
// 			Receiver: receiverIdx,
// 			Amount:   big.NewInt(int64(amount)),
// 		},
// 		btcTxId,
// 	)

// 	txHash, err := env.Aptosman.Mint(params)
// 	if err != nil {
// 		logger.Fatal(err)
// 	}

// 	return txHash, params
// }
