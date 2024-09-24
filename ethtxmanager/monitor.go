package ethtxmanager

import (
	"context"
	"errors"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
)

var (
	ErrRemoveMonitoredTx  = errors.New("failed to remove monitored tx after mined")
	ErrTransactionReceipt = errors.New("failed to get transaction receipt")
	ErrOnCanonicalChain   = errors.New("failed to check if on canonical chain")
	ErrHeaderByNumber     = errors.New("failed to get latest block header by number")
	ErrHeaderByHash       = errors.New("failed to get latest block header by hash")
)

// monitor monitors the tx until it is mined or timeout
// Monitoring procedure:
// 1. Check if the tx is mined, if mined, remove it from db and return
// 2. Check if the sentAfter block is on canonical chain, if yes, remove it from db and return
// 3. Check if the tx is timeout for monitoring, if yes, remove it from db and return
func (txmgr *EthTxManager) monitor(mtx *monitoredTx, ctx context.Context) error {
	newLogger := logger.WithFields(
		"txHash", mtx.TxHash.String(),
		"requestTxHash", mtx.RequestTxHash.String(),
	)

	removeMonitoredTx := func() error {
		err := txmgr.mgrdb.removeMonitoredTxAfterMined(mtx.TxHash)
		if err != nil {
			newLogger.Error("failed to remove monitored tx after mined: err=%v", err)
			return ErrRemoveMonitoredTx
		}
		newLogger.Debug("removed monitored tx after mined")
		return nil
	}

	// get transaction receipt
	receipt, err := txmgr.etherman.Client().TransactionReceipt(ctx, mtx.TxHash)
	if err != nil && err.Error() != ErrMsgNotFound {
		newLogger.Errorf("failed to get transaction receipt: err=%v", err)
		return ErrTransactionReceipt
	}

	// if the tx is mined, remove it from db
	if receipt != nil && receipt.BlockNumber != nil {
		return removeMonitoredTx()
	}

	// check if the sentAfter block is on canonical chain
	ok, err := txmgr.etherman.OnCanonicalChain(mtx.SentAfter)
	if err != nil {
		newLogger.Errorf("failed to check if on canonical chain: err=%v", err)
		return ErrOnCanonicalChain
	}
	// if not, remove the tx from db
	if !ok {
		newLogger.Debug("tx is not on canonical chain")
		return removeMonitoredTx()
	}

	// check timeout for monitoring the tx
	sentAfter, err := txmgr.etherman.Client().HeaderByHash(ctx, mtx.SentAfter)
	if err != nil {
		newLogger.Errorf("failed to get sentAfter block: err=%v", err)
		return ErrHeaderByHash
	}
	latest, err := txmgr.etherman.Client().HeaderByNumber(ctx, nil)
	if err != nil {
		newLogger.Errorf("failed to get latest block: err=%v", err)
		return ErrHeaderByNumber
	}

	diff := latest.Number.Uint64() - sentAfter.Number.Uint64()
	if diff > txmgr.cfg.TimeoutOnMonitoringPendingTxs {
		newLogger.Errorf("tx has not been mined for %d blocks", txmgr.cfg.TimeoutOnMonitoringPendingTxs)
		return removeMonitoredTx()
	}

	return nil
}
