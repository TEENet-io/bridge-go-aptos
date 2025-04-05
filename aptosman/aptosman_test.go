package aptosman

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// import (
// 	"crypto/ed25519"
// 	"testing"

// 	// "github.com/TEENet-io/bridge-go/common"
// 	"fmt"

// 	"github.com/TEENet-io/bridge-go/multisig_client"
// 	"github.com/aptos-labs/aptos-go-sdk"
// 	"github.com/stretchr/testify/assert"
// )

// var TEST_APTOS_ACCOUNTS = GenPrivateKeys(3)                                                           // 生成3个测试账户
// var TEST_ADMIN_ACCOUNT_ADDRESS = "0xa55ec7c0295b4a56c19e00778c1606eb51ca425b9c0e9107d7373b91469553a4" // 测试管理员账户地址
// var TEST_ADMIN_ACCOUNT = GenPrivateKeys(1)                                                            // 生成1个测试管理员账户
// var ss, _ = multisig_client.NewRandomLocalSchnorrSigner()

// func TestNonce(t *testing.T) {
// 	env, err := NewSimAptosman(TEST_APTOS_ACCOUNTS, ss)
// 	assert.NoError(t, err)
// 	fmt.Println("success Create SimAptosman")

// 	// fund account
// 	err = env.Aptosman.aptosClient.Fund(env.Aptosman.account.AccountAddress(), 100000000)
// 	assert.NoError(t, err)
// 	fmt.Println("success Fund account")

// 	// register account intwbtc
// 	txHash, err := env.Aptosman.RegisterTWBTC(env.Aptosman.account)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// 	fmt.Println("success Register account intwbtc")

// 	// 在模拟环境中执行铸币操作
// 	txHash, err = env.Mint([]byte("btc_tx_id_1"), "0", 100)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// 	fmt.Println("success Mint TxHash:", txHash)

// 	// 获取账户序列号（相当于nonce）

// 	account, err := env.Aptosman.aptosClient.Account(env.Aptosman.account.AccountAddress())
// 	assert.NoError(t, err)
// 	sequenceNumber, err := account.SequenceNumber()
// 	assert.NoError(t, err)
// 	assert.Equal(t, uint64(0x2), sequenceNumber)
// 	fmt.Println("success Get sequence number")
// 	// 执行铸币操作，使用admin账户
// 	txHash, err = env.Aptosman.MintTokensToContract(1000000000000000000)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// 	fmt.Println("success MintTokensToContract")

// 	// txHash, err = env.Aptosman.TWBTCApprove(1)
// 	// assert.NoError(t, err)
// 	// assert.NotEmpty(t, txHash)
// 	// fmt.Println("success TWBTCApprove")
// 	// 确认序列号增加
// 	account, err = env.Aptosman.aptosClient.Account(env.Aptosman.account.AccountAddress())
// 	assert.NoError(t, err)
// 	sequenceNumber, err = account.SequenceNumber()
// 	assert.NoError(t, err)
// 	assert.Equal(t, uint64(0x3), sequenceNumber)
// 	fmt.Println("success Get sequence number")
// }

// func TestIsPrepared(t *testing.T) {

// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)
// 	aptosman := env.Aptosman

// 	// 准备参数
// 	params := &PrepareParams{
// 		RequestTxHash: "req_tx_hash_2aaaaaaa",
// 		Requester:     "0x123", // 示例账户地址
// 		Receiver:      "bc1q123xyz",
// 		Amount:        uint64(100),
// 		OutpointTxIds: []string{"tx_id_12", "tx_id_22"},
// 		OutpointIdxs:  []uint16{0, 1},
// 	}

// 	// 执行赎回准备
// 	txHash, err := aptosman.RedeemPrepare(aptosman.account, params)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// 	fmt.Println("success RedeemPrepare")

//		// 验证赎回是否已准备
//		prepared, err := aptosman.IsPrepared(params.RequestTxHash)
//		assert.NoError(t, err)
//		assert.True(t, prepared)
//	}
var TEST_ADMIN_ACCOUNT_ADDRESS = "0x26f032ddd97e788550f65b8d20f9d037c4330fa27f6f92247f55bd11940774ed"

func TestRedeemRequest(t *testing.T) {
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	aptosman := env.Aptosman

	requestParams := &RequestParams{
		Amount:   uint64(1000000),
		Receiver: "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih",
	}

	txHash, err := aptosman.RedeemRequest(env.Aptosman.account, requestParams)
	assert.NoError(t, err)
	assert.NotEmpty(t, txHash)
	fmt.Println("success RedeemRequest")
	fmt.Println("txHash:", txHash)

}

// func TestIsMinted(t *testing.T) {
// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)
// 	aptosman := env.Aptosman

// 	// 准备铸币参数
// 	btcTxId := []byte("02723a1c68c90aef65704cc789fb574122f6edbbc0aa96e997a09e071df72ae1")
// 	params := &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   uint64(100000000),
// 		Receiver: "0xdbf1767f53d52d319843800c63c5e32d66411864", // 示例接收者地址
// 	}

// 	// 执行铸币
// 	txHash, err := aptosman.Mint(params)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// 	logger.WithField("txHash", txHash).Info("txHash")

// 	// 验证铸币状态
// 	minted, err := aptosman.IsMinted(string(btcTxId))
// 	assert.NoError(t, err)
// 	assert.True(t, minted)
// 	fmt.Println("success IsMinted")
// }

// func TestGetEventLogs(t *testing.T) {
// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)

// 	// 执行铸币操作
// 	btcTxId := []byte("mint_event_test")
// 	mintParams := &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   uint64(1000),
// 		Receiver: "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", // 示例接收者地址
// 	}
// 	_, err = env.Aptosman.Mint(mintParams)
// 	assert.NoError(t, err)

// 	// 执行赎回准备操作
// 	prepareParams := &PrepareParams{
// 		RequestTxHash: "prepare_event_aaaatest",
// 		Requester:     "0x123",
// 		Receiver:      "bc1q456abc",
// 		Amount:        uint64(400),
// 		OutpointTxIds: []string{"tx_12", "tx_22"},
// 		OutpointIdxs:  []uint16{0, 1},
// 	}
// 	_, err = env.Aptosman.RedeemPrepare(env.Aptosman.account, prepareParams)
// 	assert.NoError(t, err)

// 	// 获取当前版本号
// 	version, err := env.Aptosman.GetLatestFinalizedVersion()
// 	assert.NoError(t, err)

// 	// 获取事件日志
// 	mintEvents, requestEvents, prepareEvents, err := env.Aptosman.GetModuleEvents(0, version)
// 	// assert.NoError(t, err)
// 	// assert.GreaterOrEqual(t, len(mintEvents), 1)
// 	// assert.Equal(t, len(requestEvents), 0)
// 	// assert.GreaterOrEqual(t, len(prepareEvents), 1)

// 	// 验证铸币事件
// 	found := false
// 	for _, event := range mintEvents {
// 		if event.BtcTxId == string(btcTxId) {
// 			assert.Equal(t, uint64(1000), event.Amount)
// 			assert.Equal(t, "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", event.Receiver)
// 			found = true
// 			break
// 		}
// 	}
// 	assert.True(t, found, "Mint event not found")

// 	// 验证赎回准备事件
// 	found = false
// 	for _, event := range prepareEvents {
// 		if event.RequestTxHash == "prepare_event_test" {
// 			assert.Equal(t, uint64(400), event.Amount)
// 			assert.Equal(t, "0x123", event.Requester)
// 			assert.Equal(t, "bc1q456abc", event.Receiver)
// 			assert.Len(t, event.OutpointTxIds, 2)
// 			assert.Len(t, event.OutpointIdxs, 2)
// 			found = true
// 			break
// 		}
// 	}
// 	assert.True(t, found, "Prepare event not found")

// 	// 创建一个用户账户
// 	userAccount, err := NewAccount(TEST_APTOS_ACCOUNTS[1])
// 	assert.NoError(t, err)

// 	// 注册用户账户
// 	// 此处假设有一个RegisterAccount方法
// 	err = env.RegisterAccount(userAccount)
// 	assert.NoError(t, err)

// 	// 执行赎回请求
// 	requestParams := &RequestParams{
// 		Amount:   uint64(80),
// 		Receiver: "bc1q123xyz",
// 	}
// 	_, err = env.Aptosman.RedeemRequest(userAccount, requestParams)
// 	assert.NoError(t, err)

// 	// 再次获取事件
// 	version, err = env.Aptosman.GetLatestFinalizedVersion()
// 	assert.NoError(t, err)
// 	_, requestEvents, _, err = env.Aptosman.GetModuleEvents(0, version)
// 	assert.NoError(t, err)
// 	assert.GreaterOrEqual(t, len(requestEvents), 1)

// 	// 验证赎回请求事件
// 	found = false
// 	for _, event := range requestEvents {
// 		if event.Amount == uint64(80) {
// 			assert.Equal(t, "bc1q123xyz", event.Receiver)
// 			found = true
// 			break
// 		}
// 	}
// 	assert.True(t, found, "Request event not found")
// }

// func TestRedeemPrepare(t *testing.T) {
// 	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
// 	assert.NoError(t, err)
// 	aptosman := env.Aptosman

// 	// 准备参数
// 	params := &PrepareParams{
// 		RequestTxHash: "req_tx_hash_1",
// 		Requester:     "0x123",
// 		Receiver:      "bc1q123xyz",
// 		Amount:        uint64(100),
// 		OutpointTxIds: []string{"tx_id_12", "tx_id_22", "tx_id_32"},
// 		OutpointIdxs:  []uint16{0, 1, 2},
// 	}

// 	// 执行赎回准备
// 	txHash, err := aptosman.RedeemPrepare(aptosman.account, params)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, txHash)
// }

// // NewSimAptosman 创建新的模拟Aptosman测试环境
// func NewSimAptosman(privateKeys []ed25519.PrivateKey, signer multisig_client.SchnorrSigner) (*SimAptosman, error) {
// 	// 创建账户
// 	account, err := NewAccount(privateKeys[0])
// 	if err != nil {
// 		return nil, err
// 	}
// 	// 创建Aptosman配置
// 	cfg := &AptosmanConfig{
// 		URL:           "https://fullnode.devnet.aptoslabs.com", // 使用本地模拟环境URL
// 		ModuleAddress: "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864",
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

// func (s *SimAptosman) Mint(btcTxId []byte, receiver string, amount uint64) (string, error) {
// 	params := &MintParams{
// 		BtcTxId:  btcTxId,
// 		Amount:   amount,
// 		Receiver: receiver,
// 	}
// 	return s.Aptosman.Mint(params)
// }

// // RegisterAccount 注册账户以使用TWBTC
// func (s *SimAptosman) RegisterAccount(account *aptos.Account) error {
// 	// 模拟注册账户逻辑
// 	// 实际实现应该调用注册账户的合约函数
// 	return nil
// }

// // TransferTWBTC 转移TWBTC到指定账户
// func (s *SimAptosman) TransferTWBTC(receiver aptos.AccountAddress, amount uint64) error {
// 	// 模拟转账逻辑
// 	// 实际实现应该调用转账的合约函数

// 	return nil
// }
