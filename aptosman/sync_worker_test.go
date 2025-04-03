package aptosman

import (
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"
)

// var TEST_ADMIN_ACCOUNT_ADDRESS = "0xa55ec7c0295b4a56c19e00778c1606eb51ca425b9c0e9107d7373b91469553a4"

func TestSyncWorker(t *testing.T) {
	// 创建模拟环境
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	assert.NotNil(t, env)

	// 创建同步工作器
	worker := NewAptosSyncWorker(env.Aptosman)
	assert.NotNil(t, worker)

	// 测试获取最新版本号
	t.Run("Test GetNewestLedgerFinalizedNumber", func(t *testing.T) {
		version, err := worker.GetNewestLedgerFinalizedNumber()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.True(t, version.Cmp(big.NewInt(0)) >= 0)
	})

	// 测试获取事件
	t.Run("Test GetTimeOrderedEvents", func(t *testing.T) {
		// 首先创建一些事件
		// 1. 铸币事件
		btcTxId := common.RandBytes32()
		mintParams := &MintParams{
			BtcTxId:  btcTxId[:],
			Amount:   uint64(1000000),
			Receiver: "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d",
		}
		txHash, err := env.Aptosman.Mint(mintParams)
		assert.NoError(t, err)
		assert.NotEmpty(t, txHash)

		// 2. 赎回请求事件
		requestParams := &RequestParams{
			Amount:   uint64(500000),
			Receiver: "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
		}
		txHash, err = env.Aptosman.RedeemRequest(env.Accounts[1], requestParams)
		assert.NoError(t, err)
		assert.NotEmpty(t, txHash)

		// 3. 赎回准备事件
		prepareParams := &PrepareParams{
			RequestTxHash: "0x123",
			Requester:     "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d",
			Receiver:      "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			Amount:        uint64(500000),
			OutpointTxIds: []string{"0x456", "0x789"},
			OutpointIdxs:  []uint16{0, 1},
		}
		txHash, err = env.Aptosman.RedeemPrepare(env.Aptosman.account, prepareParams)
		assert.NoError(t, err)
		assert.NotEmpty(t, txHash)

		// 获取当前版本号
		currentVersion, err := worker.GetNewestLedgerFinalizedNumber()
		assert.NoError(t, err)

		// 获取从0到当前版本的所有事件
		minted, requested, prepared, err := worker.GetTimeOrderedEvents(
			big.NewInt(0),
			currentVersion,
		)
		assert.NoError(t, err)

		// 验证铸币事件
		assert.NotEmpty(t, minted)
		found := false
		for _, ev := range minted {
			if ev.BtcTxId == btcTxId {
				found = true
				assert.Equal(t, big.NewInt(1000000), ev.Amount)
				assert.Equal(t, "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d", ev.Receiver)
				break
			}
		}
		assert.True(t, found, "Mint event not found")

		// 验证赎回请求事件
		assert.NotEmpty(t, requested)
		found = false
		for _, ev := range requested {
			if ev.Amount.Cmp(big.NewInt(500000)) == 0 {
				found = true
				assert.Equal(t, "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq", ev.Receiver)
				assert.True(t, ev.IsValidReceiver)
				break
			}
		}
		assert.True(t, found, "Request event not found")

		// 验证赎回准备事件
		assert.NotEmpty(t, prepared)
		found = false
		for _, ev := range prepared {
			if ev.Amount.Cmp(big.NewInt(500000)) == 0 {
				found = true
				assert.Equal(t, "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq", ev.Receiver)
				assert.Equal(t, "0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d", ev.Requester)
				assert.Equal(t, 2, len(ev.OutpointTxIds))
				assert.Equal(t, 2, len(ev.OutpointIdxs))
				break
			}
		}
		assert.True(t, found, "Prepare event not found")
	})

	// 测试事件顺序
	t.Run("Test Events Order", func(t *testing.T) {
		currentVersion, err := worker.GetNewestLedgerFinalizedNumber()
		assert.NoError(t, err)

		// 获取事件
		minted, _, _, err := worker.GetTimeOrderedEvents(
			big.NewInt(0),
			currentVersion,
		)
		assert.NoError(t, err)

		// 验证事件是按时间顺序排列的
		if len(minted) > 1 {
			for i := 1; i < len(minted); i++ {
				// 验证版本号是递增的
				prevTx := common.HexStrToBytes32(minted[i-1].MintTxHash.String())
				currTx := common.HexStrToBytes32(minted[i].MintTxHash.String())
				assert.True(t, string(prevTx[:]) <= string(currTx[:]))
			}
		}

		// 类似地验证其他事件类型的顺序
		// ... 可以根据需要添加更多验证
	})

	// 测试版本号范围
	t.Run("Test Version Range", func(t *testing.T) {
		// 测试无效的版本号范围
		minted, requested, prepared, err := worker.GetTimeOrderedEvents(
			big.NewInt(100),
			big.NewInt(50),
		)
		assert.Error(t, err)
		assert.Nil(t, minted)
		assert.Nil(t, requested)
		assert.Nil(t, prepared)

		// 测试相同的版本号
		minted, requested, prepared, err = worker.GetTimeOrderedEvents(
			big.NewInt(50),
			big.NewInt(50),
		)
		assert.NoError(t, err)
		// 可能没有事件，但不应该报错
	})
}
