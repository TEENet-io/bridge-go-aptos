package aptosman

import (
	"crypto/ed25519"
	"testing"

	// "github.com/TEENet-io/bridge-go/common"
	"fmt"

	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/stretchr/testify/assert"
)

var TEST_APTOS_ACCOUNTS = GenPrivateKeys(3)                                                           // 生成3个测试账户
var TEST_ADMIN_ACCOUNT_ADDRESS = "0xa55ec7c0295b4a56c19e00778c1606eb51ca425b9c0e9107d7373b91469553a4" // 测试管理员账户地址
var TEST_ADMIN_ACCOUNT = GenPrivateKeys(1)                                                            // 生成1个测试管理员账户
var ss, _ = multisig_client.NewRandomLocalSchnorrSigner()

func TestNonce(t *testing.T) {
	env, err := NewSimAptosman(TEST_APTOS_ACCOUNTS, ss)
	assert.NoError(t, err)
	fmt.Println("success Create SimAptosman")

	// fund account
	err = env.Aptosman.aptosClient.Fund(env.Aptosman.account.AccountAddress(), 100000000)
	assert.NoError(t, err)
	fmt.Println("success Fund account")

	// register account intwbtc
	txHash, err := env.Aptosman.RegisterTWBTC(env.Aptosman.account)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success Register account intwbtc")

	// 在模拟环境中执行铸币操作
	txHash, err = env.Mint([]byte("btc_tx_id_1"), "0", 100)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success Mint")

	// 获取账户序列号（相当于nonce）

	account, err := env.Aptosman.aptosClient.Account(env.Aptosman.account.AccountAddress())
	assert.NoError(t, err)
	sequenceNumber, err := account.SequenceNumber()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x2), sequenceNumber)
	fmt.Println("success Get sequence number")
	// 执行铸币操作，使用admin账户
	txHash, err = env.Aptosman.MintTokensToContract(1000000000000000000)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success MintTokensToContract")

	// txHash, err = env.Aptosman.TWBTCApprove(1)
	// assert.NoError(t, err)
	// assert.NotEmpty(t, txHash)
	// fmt.Println("success TWBTCApprove")
	// 确认序列号增加
	account, err = env.Aptosman.aptosClient.Account(env.Aptosman.account.AccountAddress())
	assert.NoError(t, err)
	sequenceNumber, err = account.SequenceNumber()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0x3), sequenceNumber)
	fmt.Println("success Get sequence number")
}

func TestIsPrepared(t *testing.T) {

	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	aptosman := env.Aptosman

	// 准备参数
	params := &PrepareParams{
		RequestTxHash: "req_tx_hash_2aaaaaaa",
		Requester:     "0x123", // 示例账户地址
		Receiver:      "bc1q123xyz",
		Amount:        uint64(100),
		OutpointTxIds: []string{"tx_id_12", "tx_id_22"},
		OutpointIdxs:  []uint16{0, 1},
	}

	// 执行赎回准备
	txHash, err := aptosman.RedeemPrepare(aptosman.account, params)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success RedeemPrepare")

	// 验证赎回是否已准备
	prepared, err := aptosman.IsPrepared(params.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, prepared)
}

func TestIsMinted(t *testing.T) {
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	aptosman := env.Aptosman

	// 准备铸币参数
	btcTxId := []byte("btc_tx_id_tesassadsaaaaat")
	params := &MintParams{
		BtcTxId:  btcTxId,
		Amount:   uint64(100),
		Receiver: "0x456", // 示例接收者地址
	}
	fmt.Println("success Prepare MintParams")

	// 执行铸币
	txHash, err := aptosman.Mint(params)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success Mint")

	// 验证铸币状态
	minted, err := aptosman.IsMinted(string(btcTxId))
	assert.NoError(t, err)
	assert.True(t, minted)
	fmt.Println("success IsMinted")
}

func TestGetEventLogs(t *testing.T) {
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)

	// 执行铸币操作
	btcTxId := []byte("mint_event_test")
	mintParams := &MintParams{
		BtcTxId:  btcTxId,
		Amount:   uint64(1000),
		Receiver: "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d", // 示例接收者地址
	}
	_, err = env.Aptosman.Mint(mintParams)
	assert.NoError(t, err)

	// 执行赎回准备操作
	prepareParams := &PrepareParams{
		RequestTxHash: "prepare_event_aaaatest",
		Requester:     "0x123",
		Receiver:      "bc1q456abc",
		Amount:        uint64(400),
		OutpointTxIds: []string{"tx_12", "tx_22"},
		OutpointIdxs:  []uint16{0, 1},
	}
	_, err = env.Aptosman.RedeemPrepare(env.Aptosman.account, prepareParams)
	assert.NoError(t, err)

	// 获取当前版本号
	version, err := env.Aptosman.GetLatestFinalizedVersion()
	assert.NoError(t, err)

	// 获取事件日志
	mintEvents, requestEvents, prepareEvents, err := env.Aptosman.GetModuleEvents(0, version)
	// assert.NoError(t, err)
	// assert.GreaterOrEqual(t, len(mintEvents), 1)
	// assert.Equal(t, len(requestEvents), 0)
	// assert.GreaterOrEqual(t, len(prepareEvents), 1)

	// 验证铸币事件
	found := false
	for _, event := range mintEvents {
		if event.BtcTxId == string(btcTxId) {
			assert.Equal(t, uint64(1000), event.Amount)
			assert.Equal(t, "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d", event.Receiver)
			found = true
			break
		}
	}
	assert.True(t, found, "Mint event not found")

	// 验证赎回准备事件
	found = false
	for _, event := range prepareEvents {
		if event.RequestTxHash == "prepare_event_test" {
			assert.Equal(t, uint64(400), event.Amount)
			assert.Equal(t, "0x123", event.Requester)
			assert.Equal(t, "bc1q456abc", event.Receiver)
			assert.Len(t, event.OutpointTxIds, 2)
			assert.Len(t, event.OutpointIdxs, 2)
			found = true
			break
		}
	}
	assert.True(t, found, "Prepare event not found")

	// 创建一个用户账户
	userAccount, err := NewAccount(TEST_APTOS_ACCOUNTS[1])
	assert.NoError(t, err)

	// 注册用户账户
	// 此处假设有一个RegisterAccount方法
	err = env.RegisterAccount(userAccount)
	assert.NoError(t, err)

	// 执行赎回请求
	requestParams := &RequestParams{
		Amount:   uint64(80),
		Receiver: "bc1q123xyz",
	}
	_, err = env.Aptosman.RedeemRequest(userAccount, requestParams)
	assert.NoError(t, err)

	// 再次获取事件
	version, err = env.Aptosman.GetLatestFinalizedVersion()
	assert.NoError(t, err)
	_, requestEvents, _, err = env.Aptosman.GetModuleEvents(0, version)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(requestEvents), 1)

	// 验证赎回请求事件
	found = false
	for _, event := range requestEvents {
		if event.Amount == uint64(80) {
			assert.Equal(t, "bc1q123xyz", event.Receiver)
			found = true
			break
		}
	}
	assert.True(t, found, "Request event not found")
}

func TestRedeemPrepare(t *testing.T) {
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	aptosman := env.Aptosman

	// 准备参数
	params := &PrepareParams{
		RequestTxHash: "req_tx_hash_1",
		Requester:     "0x123",
		Receiver:      "bc1q123xyz",
		Amount:        uint64(100),
		OutpointTxIds: []string{"tx_id_12", "tx_id_22", "tx_id_32"},
		OutpointIdxs:  []uint16{0, 1, 2},
	}

	// 执行赎回准备
	txHash, err := aptosman.RedeemPrepare(aptosman.account, params)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
}

// func TestMint(t *testing.T) {
// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)
// 	aptosman := env.Aptosman

// 	TestAccount := "0xa2184ca989d7741a463c0a77f0283aab71c86410f6596a814b217329573488ee"
// 	balance, err := aptosman.GetTWBTCBalance(TestAccount)
// 	assert.NoError(t, err)
// 	assert.Equal(t, uint64(0), balance)
// 	// 准备铸币参数
// 	btcTxId := []byte("mint_test_tx_id")
// 	params := &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   uint64(100),
// 		Receiver: TestAccount,
// 	}

// 	// 执行铸币
// 	txHash, err := aptosman.Mint(params)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)

// 	// 检查接收者余额
// 	balance, err := aptosman.GetTWBTCBalance(TestAccount)
// 	assert.NoError(t, err)
// 	assert.Equal(t, uint64(100), balance)
// }

// // PASSED
// func TestGetLatestFinalizedVersion(t *testing.T) {
// 	// 配置用于测试的Aptosman
// 	cfg := &AptosmanConfig{
// 		URL:           "https://fullnode.devnet.aptoslabs.com",
// 		ModuleAddress: "0x1",
// 		Network:       "devnet",
// 	}

// 	// 使用测试账户
// 	account, err := NewAccount(TEST_APTOS_ACCOUNTS[0])
// 	assert.NoError(t, err)

// 	// 创建Aptosman实例
// 	aptosman, err := NewAptosman(cfg, account)
// 	if err != nil {
// 		t.Skip("无法连接到Aptos网络，跳过测试:", err)
// 		return
// 	}

// 	version, err := aptosman.GetLatestFinalizedVersion()
// 	assert.NoError(t, err)
// 	assert.NotZero(t, version)
// }

// // PASSED
// func TestDebugGetLatestFinalizedVersion(t *testing.T) {
// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)
// 	aptosman := env.Aptosman

// 	common.Debug = true
// 	defer func() {
// 		common.Debug = false
// 	}()

// 	// 获取当前版本
// 	version, err := aptosman.GetLatestFinalizedVersion()
// 	assert.NoError(t, err)
// 	assert.NotZero(t, version)

// 	// 执行一些操作推进版本
// 	btcTxId := []byte("debug_test_tx_id")
// 	params := &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   uint64(100),
// 		Receiver: "0x123",
// 	}
// 	_, err = aptosman.Mint(params)
// 	assert.NoError(t, err)

// 	// 获取新版本并验证它已增加
// 	newVersion, err := aptosman.GetLatestFinalizedVersion()
// 	assert.NoError(t, err)
// 	assert.Greater(t, newVersion, version)
// }

// NewSimAptosman 创建新的模拟Aptosman测试环境
func NewSimAptosman(privateKeys []ed25519.PrivateKey, signer multisig_client.SchnorrSigner) (*SimAptosman, error) {
	// 创建账户
	account, err := NewAccount(privateKeys[0])
	if err != nil {
		return nil, err
	}
	// 创建Aptosman配置
	cfg := &AptosmanConfig{
		URL:           "https://fullnode.devnet.aptoslabs.com", // 使用本地模拟环境URL
		ModuleAddress: "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d",
		Network:       "devnet",
	}

	// 创建Aptosman实例
	aptosman, err := NewAptosman(cfg, account)
	if err != nil {
		return nil, err
	}

	return &SimAptosman{
		Aptosman: aptosman,
	}, nil
}

// func NewSimAptosman_from_privateKey(privateKey string) (*SimAptosman, error) {
// 	account, err := createAccountFromPrivateKey(privateKey)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// 创建Aptosman配置
// 	cfg := &AptosmanConfig{
// 		URL:           "https://fullnode.devnet.aptoslabs.com", // 使用本地模拟环境URL
// 		ModuleAddress: "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d",
// 		Network:       "devnet",
// 	}

// 	// 创建Aptosman实例
// 	aptosman, err := NewAptosman(cfg, account)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &SimAptosman{
// 		Aptosman: aptosman,
// 	}, nil
// }

// Mint 在模拟环境中执行铸币操作
func (s *SimAptosman) Mint(btcTxId []byte, receiver string, amount uint64) (string, error) {
	params := &MintParams{
		BtcTxId:  btcTxId,
		Amount:   amount,
		Receiver: receiver,
	}
	return s.Aptosman.Mint(params)
}

// RegisterAccount 注册账户以使用TWBTC
func (s *SimAptosman) RegisterAccount(account *aptos.Account) error {
	// 模拟注册账户逻辑
	// 实际实现应该调用注册账户的合约函数
	return nil
}

// TransferTWBTC 转移TWBTC到指定账户
func (s *SimAptosman) TransferTWBTC(receiver aptos.AccountAddress, amount uint64) error {
	// 模拟转账逻辑
	// 实际实现应该调用转账的合约函数

	return nil
}

// 以下是测试辅助函数

// currentVersionNum 获取当前区块链版本号
func currentVersionNum(t *testing.T, env *SimAptosman) uint64 {
	version, err := env.Aptosman.GetLatestFinalizedVersion()
	assert.NoError(t, err)
	return version
}

// checkMintedEvent 验证铸币事件
func checkMintedEvent(t *testing.T, ev *MintedEvent, params *MintParams) {
	assert.Equal(t, string(params.BtcTxId), ev.BtcTxId)
	assert.Equal(t, params.Receiver, ev.Receiver)
	assert.Equal(t, params.Amount, ev.Amount)
}

// checkPreparedEvent 验证赎回准备事件
func checkPreparedEvent(t *testing.T, ev *RedeemPreparedEvent, params *PrepareParams) {
	assert.Equal(t, params.RequestTxHash, ev.PrepareTxHash)
	assert.Equal(t, params.Requester, ev.Requester)
	assert.Equal(t, params.Amount, ev.Amount)
	assert.Equal(t, params.OutpointTxIds, ev.OutpointTxIds)

	// 比较OutpointIdxs
	assert.Equal(t, len(params.OutpointIdxs), len(ev.OutpointIdxs))
	for i, idx := range ev.OutpointIdxs {
		assert.Equal(t, params.OutpointIdxs[i], idx)
	}
}

// checkRequestedEvent 验证赎回请求事件
func checkRequestedEvent(t *testing.T, ev *RedeemRequestedEvent, params *RequestParams) {
	assert.Equal(t, params.Amount, ev.Amount)
	assert.Equal(t, params.Receiver, ev.Receiver)
}
