package aptosman

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var TEST_ADMIN_ACCOUNT_ADDRESS = "0xa55ec7c0295b4a56c19e00778c1606eb51ca425b9c0e9107d7373b91469553a4"

func TestMgrWorker(t *testing.T) {
	// 创建模拟环境
	env, err := NewSimAptosman_from_privateKey(TEST_ADMIN_ACCOUNT_ADDRESS)
	assert.NoError(t, err)
	assert.NotNil(t, env)

	// 创建 MgrWorker
	worker := NewAptosMgrWorker(env.Aptosman)
	assert.NotNil(t, worker)

	// 测试 GetLatestLedgerNumber
	t.Run("Test GetLatestLedgerNumber", func(t *testing.T) {
		version, err := worker.GetLatestLedgerNumber()
		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.True(t, version.Cmp(big.NewInt(0)) >= 0)
	})

	// 测试铸币流程
	t.Run("Test Mint Flow", func(t *testing.T) {
		// 准备铸币参数
		btcTxId := common.RandBytes32()
		mintParam := &agreement.MintParameter{
			BtcTxId:  btcTxId,
			Amount:   big.NewInt(1000000),
			Receiver: []byte("0x1"),
			Rx:       big.NewInt(1),
			S:        big.NewInt(1),
		}

		// 检查是否已铸币
		isMinted, err := worker.IsMinted(btcTxId)
		assert.NoError(t, err)
		assert.False(t, isMinted)

		// 执行铸币
		txId, _, err := worker.DoMint(mintParam)
		assert.NoError(t, err)
		assert.NotNil(t, txId)

		// 等待交易完成
		time.Sleep(2 * time.Second)

		// 检查交易状态
		status, foundVersion, err := worker.GetTxStatus(txId)
		assert.NoError(t, err)
		assert.NotEqual(t, agreement.MalForm, status)
		if status == agreement.Success {
			assert.NotNil(t, foundVersion)
		}

		// 再次检查是否已铸币
		isMinted, err = worker.IsMinted(btcTxId)
		assert.NoError(t, err)
		assert.True(t, isMinted)
	})

	// 测试赎回准备流程
	t.Run("Test Prepare Flow", func(t *testing.T) {
		// 准备赎回参数
		requestTxId := common.RandBytes32()
		prepareParam := &agreement.PrepareParameter{
			RequestTxHash: requestTxId,
			Requester:     []byte("0x1"),
			Receiver:      "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
			Amount:        big.NewInt(500000),
			OutpointTxIds: []ethcommon.Hash{
				ethcommon.BytesToHash(requestTxId[:]),
			},
			OutpointIdxs: []uint16{0, 1},
			Rx:           big.NewInt(1),
			S:            big.NewInt(1),
		}

		// 检查是否已准备
		isPrepared, err := worker.IsPrepared(requestTxId)
		assert.NoError(t, err)
		assert.False(t, isPrepared)

		// 执行准备
		txId, _, err := worker.DoPrepare(prepareParam)
		assert.NoError(t, err)
		assert.NotNil(t, txId)

		// 等待交易完成
		time.Sleep(2 * time.Second)

		// 检查交易状态
		status, foundVersion, err := worker.GetTxStatus(txId)
		assert.NoError(t, err)
		assert.NotEqual(t, agreement.MalForm, status)
		if status == agreement.Success {
			assert.NotNil(t, foundVersion)
		}

		// 再次检查是否已准备
		isPrepared, err = worker.IsPrepared(requestTxId)
		assert.NoError(t, err)
		assert.True(t, isPrepared)
	})

	// 测试交易状态
	t.Run("Test Transaction Status", func(t *testing.T) {
		testCases := []struct {
			name     string
			txId     []byte
			expected agreement.MonitoredTxStatus
		}{
			{
				name:     "Invalid Transaction Hash",
				txId:     []byte("invalid_tx_hash"),
				expected: agreement.MalForm,
			},
			{
				name:     "Non-existent Transaction",
				txId:     []byte("0x2"),
				expected: agreement.Limbo,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				status, _, err := worker.GetTxStatus(tc.txId)
				if tc.expected == agreement.MalForm {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tc.expected, status)
				}
			})
		}
	})

	// 测试超时情况
	t.Run("Test Transaction Timeout", func(t *testing.T) {
		// 创建一个模拟的长时间运行的交易
		btcTxId := common.RandBytes32()
		mintParam := &agreement.MintParameter{
			BtcTxId:  btcTxId,
			Amount:   big.NewInt(1000000),
			Receiver: []byte("0x1"),
			Rx:       big.NewInt(1),
			S:        big.NewInt(1),
		}

		txId, _, err := worker.DoMint(mintParam)
		assert.NoError(t, err)

		// 等待足够长的时间
		time.Sleep(10 * time.Second)

		// 检查状态是否超时
		status, _, err := worker.GetTxStatus(txId)
		assert.NoError(t, err)
		if status == agreement.Timeout {
			t.Log("Transaction correctly timed out")
		}
	})
}

// 可选：添加帮助函数
func waitForTxStatus(worker *AptosMgrWorker, txId []byte, expectedStatus agreement.MonitoredTxStatus, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			status, _, err := worker.GetTxStatus(txId)
			if err != nil {
				return err
			}
			if status == expectedStatus {
				return nil
			}
		}
	}
}
