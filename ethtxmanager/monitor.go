package ethtxmanager

import (
	"context"
	"errors"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrDBOpRemoveMonitoredTx      = errors.New("failed to remove monitored tx after mined")
	ErrEthermanTransactionReceipt = errors.New("failed to get transaction receipt")
	ErrEthermanHeaderByNumber     = errors.New("failed to get latest block header by number")
	ErrEthermanHeaderByHash       = errors.New("failed to get latest block header by hash")

	ErrMsgNotFound = "not found"
)

// monitor monitors the tx until it is mined or timeout
// Monitoring procedure:
// 1. Check if the tx is mined, if mined, remove it from db and return
// 2. Check if the sentAfter block is on canonical chain, if yes, remove it from db and return
// 3. Check if the tx is timeout for monitoring, if yes, remove it from db and return
func (txmgr *EthTxManager) monitor(ctx context.Context, mtx *monitoredTx) error {
	newLogger := logger.WithFields(
		"txHash", mtx.TxHash.String(),
		"Id", mtx.Id.String(),
	)

	removeMonitoredTx := func(txHash ethcommon.Hash) error {
		err := txmgr.mgrdb.RemoveMonitoredTx(txHash)
		if err != nil {
			newLogger.Error("failed to remove monitored tx after mined: err=%v", err)
			return ErrDBOpRemoveMonitoredTx
		}
		newLogger.Debug("removed monitored tx after mined")
		return nil
	}

	// get transaction receipt
	receipt, err := txmgr.etherman.Client().TransactionReceipt(ctx, mtx.TxHash)
	if err != nil && err.Error() != ErrMsgNotFound {
		newLogger.Errorf("failed to get transaction receipt: err=%v", err)
		return ErrEthermanTransactionReceipt
	}

	// if the tx is mined, remove it from db
	if receipt != nil && receipt.BlockNumber != nil {
		newLogger.Debug("tx has been mined")
		return removeMonitoredTx(mtx.TxHash)
	}

	// check timeout for monitoring the tx
	sentAfter, err := txmgr.etherman.Client().HeaderByHash(ctx, mtx.SentAfter)
	if err != nil {
		newLogger.Errorf("failed to get sentAfter block: err=%v", err)
		return ErrEthermanHeaderByHash
	}
	latest, err := txmgr.etherman.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		newLogger.Errorf("failed to get latest block: err=%v", err)
		return ErrEthermanHeaderByNumber
	}

	diff := latest.Number.Uint64() - sentAfter.Number.Uint64()
	if diff > txmgr.cfg.TimeoutOnMonitoringPendingTxs {
		newLogger.Errorf("tx has not been mined for %d blocks", txmgr.cfg.TimeoutOnMonitoringPendingTxs)
		return removeMonitoredTx(mtx.TxHash)
	}

	return nil
}
