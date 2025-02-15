package ethtxmanager

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrDBOpRemoveMonitoredTx       = errors.New("failed to remove monitored tx after mined")
	ErrDBOpUpdateMonitoredTxStatus = errors.New("failed to update monitored tx status")
	ErrEthermanTransactionReceipt  = errors.New("failed to get transaction receipt")
	ErrEthermanHeaderByNumber      = errors.New("failed to get latest block header by number")
	ErrEthermanHeaderByHash        = errors.New("failed to get latest block header by hash")
)

// monitor monitors the tx until it is mined or timeout
// Monitoring procedure:
// 1. Check if the tx is mined, if mined, update its status to either "success" or "reverted"
// 2. Check if the tx is timeout for monitoring, if yes, update its status to "timeout"
func (txmgr *EthTxManager) monitorPendingTxs(ctx context.Context, mtx *MonitoredTx) error {
	newLogger := logger.WithFields(logger.Fields{
		"txHash": mtx.TxHash.String(),
		"Id":     mtx.Id.String(),
	})

	// get transaction receipt
	receipt, err := txmgr.etherman.Client().TransactionReceipt(ctx, mtx.TxHash)
	if err != nil && err.Error() != ethereum.NotFound.Error() {
		newLogger.Errorf("failed to get transaction receipt: err=%v", err)
		return ErrEthermanTransactionReceipt
	}

	// if the tx is mined, remove it from db
	if receipt != nil && receipt.BlockNumber != nil {
		newLogger.Debug("evm tx has been mined")

		var status MonitoredTxStatus
		if receipt.Status == 0 {
			newLogger.Error("evm tx mined but reverted")
			status = Reverted
		} else {
			newLogger.Debug("evm tx mined and successfully executed")
			status = Success
		}
		err := txmgr.mgrdb.UpdateMonitoredTxStatus(mtx.TxHash, status)
		if err != nil {
			newLogger.Errorf("failed to update monitored evm tx status: err=%v", err)
			return ErrDBOpUpdateMonitoredTxStatus
		}
	}

	// check timeout for monitoring the tx
	newLogger.Debugf("monitored evm tx, field SentAfter %v", mtx.SentAfter)
	newLogger.Debugf("monitored evm tx, field SentAfter.Hex() %s", mtx.SentAfter.Hex())

	sentAfter, err := txmgr.etherman.Client().HeaderByHash(ctx, mtx.SentAfter)
	newLogger.Debugf("sentAfter: %v", sentAfter)
	if err != nil {
		newLogger.Errorf("failed to get 'sentAfter' evm block: err=%v", err)
		return ErrEthermanHeaderByHash
	}
	latest, err := txmgr.etherman.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		newLogger.Errorf("failed to get latest block: err=%v", err)
		return ErrEthermanHeaderByNumber
	}

	diff := latest.Number.Uint64() - sentAfter.Number.Uint64()
	if diff > txmgr.cfg.TimeoutOnMonitoringPendingTxs {
		newLogger.Debugf("tx has not been mined for %d blocks", txmgr.cfg.TimeoutOnMonitoringPendingTxs)
		err := txmgr.mgrdb.UpdateMonitoredTxStatus(mtx.TxHash, Timeout)
		if err != nil {
			newLogger.Errorf("failed to update monitored tx status: err=%v", err)
			return ErrDBOpUpdateMonitoredTxStatus
		}
		return nil
	}

	return nil
}
