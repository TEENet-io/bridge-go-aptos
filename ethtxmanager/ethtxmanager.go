package ethtxmanager

import (
	"context"
	"errors"
	"sync"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrDBOpGetMonitoredTxs    = errors.New("failed to get monitored txs")
	ErrDBOpGetRedeemsByStatus = errors.New("failed to get redeems by status")
)

type EthTxManager struct {
	cfg           *Config
	etherman      *etherman.Etherman
	statedb       *state.StateDB
	mgrdb         *EthTxManagerDB
	schnorrWallet SchnorrThresholdWallet
	btcWallet     BtcWallet

	// public key of the schnorr threshold signature
	pubKey ethcommon.Hash

	redeemLock sync.Map
}

func New(
	cfg *Config,
	etherman *etherman.Etherman,
	statedb *state.StateDB,
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

	tickerToPrepare := time.NewTicker(txmgr.cfg.FrequencyToPrepareRedeem)
	defer tickerToPrepare.Stop()

	tickerToMonitor := time.NewTicker(txmgr.cfg.FrequencyToMonitorPendingTxs)
	defer tickerToMonitor.Stop()

	tickerToMint := time.NewTicker(txmgr.cfg.FrequencyToMint)
	defer tickerToMint.Stop()

	errCh := make(chan error, 1)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			switch err {
			case context.Canceled:
			case context.DeadlineExceeded:
			// chain interaction errors
			case ErrBridgeIsPrepared:
			case ErrBridgeRedeemPrepare:
			case ErrEthermanHeaderByNumber:
			case ErrEthermanTransactionReceipt:
			case ErrEthermanHeaderByHash:
			// statedb errors
			case ErrDBOpGetMonitoredTxByID:
			case ErrDBOpGetSignatureRequestByRequestTxHash:
			case ErrDBOpInsertMonitoredTx:
			case ErrDBOpInsertSignatureRequest:
			case ErrDBOpRemoveMonitoredTx:
			case ErrDBOpGetRedeemsByStatus:
			case ErrDBOpGetMonitoredTxs:
			// wallet errors
			case ErrBtcWalletRequest:
			case ErrSchnorrWalletSign:
			// statedb errors
			default:
				logger.Fatal(err)
			}
			return err
		case <-tickerToMint.C:
			// TODO: implement
		case <-tickerToMonitor.C:
			mtxs, err := txmgr.mgrdb.GetMonitoredTxs()
			if err != nil {
				logger.Errorf("failed to get monitored tx by status: err=%v", err)
				errCh <- ErrDBOpGetMonitoredTxs
			}

			if len(mtxs) == 0 {
				continue
			}

			wg := sync.WaitGroup{}
			wg.Add(len(mtxs))
			for _, mtx := range mtxs {
				go func() {
					defer wg.Done()
					err = txmgr.monitor(ctx, mtx)

					if err != nil {
						errCh <- err
					}
				}()
			}
			wg.Wait()
		case <-tickerToPrepare.C:
			redeemsFromDB, err := txmgr.statedb.GetRedeemsByStatus(state.RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get redeems by status: err=%v", err)
				errCh <- ErrDBOpGetRedeemsByStatus
			}

			redeems := []*state.Redeem{}
			for _, redeem := range redeemsFromDB {
				// Check whether there is any pending tx that has tried to prepare the redeem
				_, ok, err := txmgr.mgrdb.GetMonitoredTxById(redeem.RequestTxHash)
				if err != nil {
					logger.Errorf("failed to get monitored tx by request tx hash: err=%v", err)
					errCh <- ErrDBOpGetMonitoredTxByID
				}
				if !ok {
					redeems = append(redeems, redeem)
				}
			}

			if len(redeems) == 0 {
				continue
			}

			wg := sync.WaitGroup{}
			wg.Add(len(redeems))
			for _, redeem := range redeems {
				// Check whether there is a routine currently being handling the redeem
				if _, ok := txmgr.redeemLock.Load(redeem.RequestTxHash); ok {
					continue
				}

				go func() {
					defer wg.Done()
					err = txmgr.prepareRedeem(ctx, redeem)

					if err != nil {
						errCh <- err
					}
				}()
			}
			wg.Wait()
		}
	}
}
