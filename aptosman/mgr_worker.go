package aptosman

import (
	"math/big"
	"strings"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
)

// AptosMgrWorker 实现 chaintxmgr.MgrWorker 接口
type AptosMgrWorker struct {
	aptosman *Aptosman
}

// NewAptosMgrWorker 创建新的 Aptos 管理工作器
func NewAptosMgrWorker(aptosman *Aptosman) *AptosMgrWorker {
	return &AptosMgrWorker{
		aptosman: aptosman,
	}
}

// GetLatestLedgerNumber 获取最新的账本版本号
func (w *AptosMgrWorker) GetLatestLedgerNumber() (*big.Int, error) {
	version, err := w.aptosman.GetLatestFinalizedVersion()
	if err != nil {
		return nil, err
	}
	return big.NewInt(int64(version)), nil
}

// IsMinted 检查是否已经铸造
func (w *AptosMgrWorker) IsMinted(btcTxId [32]byte) (bool, error) {
	return w.aptosman.IsMinted(common.Bytes32ToHexStr(btcTxId))
}

// DoMint 执行铸币操作
func (w *AptosMgrWorker) DoMint(mint *agreement.MintParameter) ([]byte, *big.Int, error) {
	// 转换参数

	params := &MintParams{
		BtcTxId:  mint.BtcTxId[:], // 将common.Hash转换为[]byte
		Amount:   mint.Amount.Uint64(),
		Receiver: string(mint.Receiver),
	}

	// 执行铸币
	txHash, err := w.aptosman.Mint(params)
	if err != nil {
		return nil, nil, err
	}

	// 获取当前版本号
	version, err := w.GetLatestLedgerNumber()
	if err != nil {
		return []byte(txHash), nil, nil // 如果获取版本号失败，返回nil作为版本号
	}

	return []byte(txHash), version, nil
}

// IsPrepared 检查是否已经准备赎回
func (w *AptosMgrWorker) IsPrepared(requestTxId [32]byte) (bool, error) {
	return w.aptosman.IsPrepared(common.Bytes32ToHexStr(requestTxId))
}

// DoPrepare 执行赎回准备
func (w *AptosMgrWorker) DoPrepare(prepare *agreement.PrepareParameter) ([]byte, *big.Int, error) {
	// 转换 OutpointTxIds
	outpointTxIds := make([]string, len(prepare.OutpointTxIds))
	for i, txId := range prepare.OutpointTxIds {
		outpointTxIds[i] = common.Bytes32ToHexStr(txId)
	}

	// 转换参数
	params := &PrepareParams{
		RequestTxHash: common.Bytes32ToHexStr(prepare.RequestTxHash),
		Requester:     string(prepare.Requester),
		Receiver:      prepare.Receiver,
		Amount:        prepare.Amount.Uint64(),
		OutpointTxIds: outpointTxIds,
		OutpointIdxs:  prepare.OutpointIdxs,
		// Rx:            prepare.Rx,
		// S:             prepare.S,
	}

	// 执行准备
	txHash, err := w.aptosman.RedeemPrepare(w.aptosman.account, params)
	if err != nil {
		return nil, nil, err
	}

	// 获取当前版本号
	version, err := w.GetLatestLedgerNumber()
	if err != nil {
		// 如果获取版本号失败，仍然返回交易ID，但版本号为nil
		return []byte(txHash), nil, nil
	}

	return []byte(txHash), version, nil
}

// GetTxStatus 获取交易状态
func (w *AptosMgrWorker) GetTxStatus(txId []byte) (agreement.MonitoredTxStatus, *big.Int, error) {
	txHash := string(txId)

	// 设置自定义的轮询参数
	pollPeriod := 100 * time.Millisecond // 每100ms轮询一次
	pollTimeout := 5 * time.Second       // 最多等待5秒

	tx, err := w.aptosman.aptosClient.WaitForTransaction(
		txHash,
		pollPeriod,
		pollTimeout,
	)

	if err != nil {
		// 处理超时情况
		if strings.Contains(err.Error(), "timeout") {
			return agreement.Timeout, nil, nil
		}
		// 处理未找到交易的情况
		if strings.Contains(err.Error(), "not found") {
			return agreement.Limbo, nil, nil
		}
		return agreement.MalForm, nil, err
	}

	version := big.NewInt(int64(tx.Version))

	if tx.Success {
		return agreement.Success, version, nil
	}

	return agreement.Reverted, version, nil
}
