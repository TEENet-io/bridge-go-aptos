package ethtxmanager

import (
	"context"
	"errors"
	"sync"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
)

var (
	// error when the eth tx manager times out on waiting for a signature request
	ErrTimedOutOnWaitingForSignature = errors.New("timed out on waiting for signature")

	ErrInvalidSchnorrSignature = errors.New("invalid schnorr signature")

	ErrEmptyOutpointsReturned = errors.New("empty outpoints returned")

	ErrMsgNotFound = "not found"
)

type EthTxManager struct {
	ctx      context.Context
	cfg      *Config
	etherman *etherman.Etherman
	statedb  *eth2btcstate.StateDB
	mgrdb    *EthTxManagerDB

	schnorrWallet SchnorrThresholdWallet
	btcWallet     BtcWallet

	// public key of the schnorr threshold signature
	pubKey [32]byte

	redeemLock sync.Map
}

func New(
	ctx context.Context,
	cfg *Config,
	etherman *etherman.Etherman,
	statedb *eth2btcstate.StateDB,
	mgrdb *EthTxManagerDB,
	schnorrWallet SchnorrThresholdWallet,
	btcWallet BtcWallet,
) (*EthTxManager, error) {
	// Get the public key of the schnorr threshold signature
	pk, err := etherman.GetPublicKey()
	if err != nil {
		logger.Errorf("failed to get public key: err=%v", err)
		return nil, err
	}
	pubKey := common.BigInt2Bytes32(pk)

	return &EthTxManager{
		ctx:           ctx,
		etherman:      etherman,
		statedb:       statedb,
		cfg:           cfg,
		mgrdb:         mgrdb,
		pubKey:        pubKey,
		schnorrWallet: schnorrWallet,
		btcWallet:     btcWallet,
	}, nil

}

func (txmgr *EthTxManager) Start(ctx context.Context) error {
	logger.Info("starting eth tx manager")
	defer logger.Info("stopping eth tx manager")

	tickerToGetUppreparedRedeem := time.NewTicker(txmgr.cfg.FrequencyToGetUnpreparedRedeem)
	defer tickerToGetUppreparedRedeem.Stop()

	tickerToMonitorPendingTxs := time.NewTicker(txmgr.cfg.FrequencyToMonitorPendingTxs)
	defer tickerToMonitorPendingTxs.Stop()

	for {
		select {
		case <-txmgr.ctx.Done():
			return txmgr.ctx.Err()
		case <-tickerToMonitorPendingTxs.C:
			mtxs, err := txmgr.mgrdb.GetAllMonitoredTx()
			if err != nil {
				logger.Error("failed to get monitored tx by status")
				return err
			}

			if len(mtxs) == 0 {
				continue
			}

			for _, mtx := range mtxs {
				go func() {
					logger1 := logger.WithFields(
						"txHash", mtx.TxHash.String(),
						"requestTxHash", mtx.RequestTxHash.String(),
					)

					removeMonitoredTx := func() {
						err = txmgr.mgrdb.removeMonitoredTxAfterMined(mtx.TxHash)
						if err != nil {
							logger1.Error("failed to remove monitored tx after mined: err=%v", err)
							return
						}
						logger1.Debug("removed monitored tx after mined")
					}

					receipt, err := txmgr.etherman.Client().TransactionReceipt(ctx, mtx.TxHash)
					if err != nil && err.Error() != ErrMsgNotFound {
						logger1.Errorf("failed to get transaction receipt: err=%v", err)
						return
					}

					// if the tx is mined, remove it from db
					if receipt != nil && receipt.BlockNumber != nil {
						removeMonitoredTx()
						return
					}

					// check if the tx is on canonical chain
					ok, err := txmgr.etherman.OnCanonicalChain(mtx.SentAfter)
					if err != nil {
						logger1.Errorf("failed to check if on canonical chain: err=%v", err)
						return
					}
					// if the tx is not on canonical chain, remove it from db
					if !ok {
						logger1.Debug("tx is not on canonical chain")
						removeMonitoredTx()
						return
					}

					// check timeout for monitoring the tx
					sentAfter, err := txmgr.etherman.Client().HeaderByHash(ctx, mtx.SentAfter)
					if err != nil {
						logger1.Errorf("failed to get sentAfter block: err=%v", err)
					}
					latest, err := txmgr.etherman.Client().HeaderByNumber(ctx, nil)
					if err != nil {
						logger1.Errorf("failed to get latest block: err=%v", err)
						return
					}

					diff := latest.Number.Uint64() - sentAfter.Number.Uint64()
					if diff > txmgr.cfg.TimeoutOnMonitoringPendingTxs {
						logger1.Errorf("tx has not been mined for %d blocks", txmgr.cfg.TimeoutOnMonitoringPendingTxs)
						removeMonitoredTx()
						return
					}
				}()
			}
		case <-tickerToGetUppreparedRedeem.C:
			redeems, err := txmgr.statedb.GetByStatus(eth2btcstate.RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get redeems by status: err=%v", err)
				return err
			}

			for _, redeem := range redeems {
				logger1 := logger.WithFields("requestTxHash", redeem.RequestTxHash.String())

				// Check whether there is a routine currently being handling the redeem
				if _, ok := txmgr.redeemLock.Load(redeem.RequestTxHash); ok {
					continue
				}

				// Check the redeem status on bridge contract. If false, either the redeem
				// has been handled or there is a pending tx that tries to prepare the redeem.
				ok, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
				if err != nil {
					logger1.Errorf("Etherman: failed to check if prepared: err=%v", err)
					return err
				}
				if ok {
					continue
				}

				// Check whether there is any pending tx that tries to prepare the redeem
				_, ok, err = txmgr.mgrdb.GetMonitoredTxByRequestTxHash(redeem.RequestTxHash)
				if err != nil {
					logger1.Error("failed to get monitored tx by request tx hash: err=%v", err)
					return err
				}
				if ok {
					continue
				}

				go txmgr.prepareRedeem(ctx, redeem, logger1)
			}
		}
	}
}

func (txmgr *EthTxManager) Close() {
	if txmgr.mgrdb != nil {
		txmgr.mgrdb.Close()
	}
}
