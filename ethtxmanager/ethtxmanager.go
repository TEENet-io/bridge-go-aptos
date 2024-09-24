package ethtxmanager

import (
	"context"
	"sync"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type EthTxManager struct {
	ctx      context.Context
	cfg      *Config
	etherman *etherman.Etherman

	statedb *state.StateDB
	mgrdb   *EthTxManagerDB

	schnorrWallet SchnorrThresholdWallet
	btcWallet     BtcWallet

	// public key of the schnorr threshold signature
	pubKey ethcommon.Hash

	redeemLock sync.Map
}

func New(
	ctx context.Context,
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

	tickerToPrepare := time.NewTicker(txmgr.cfg.FrequencyToPrepareRedeem)
	defer tickerToPrepare.Stop()

	tickerToMonitor := time.NewTicker(txmgr.cfg.FrequencyToMonitorPendingTxs)
	defer tickerToMonitor.Stop()

	tickerToMint := time.NewTicker(txmgr.cfg.FrequencyToMint)
	defer tickerToMint.Stop()

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-txmgr.ctx.Done():
			return txmgr.ctx.Err()
		case <-tickerToMint.C:
		case <-tickerToMonitor.C:
			mtxs, err := txmgr.mgrdb.GetAllMonitoredTx()
			if err != nil {
				logger.Fatal("failed to get monitored tx by status: err=%v", err)
			}

			if len(mtxs) == 0 {
				continue
			}

			for _, mtx := range mtxs {
				wg.Add(1)
				go func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Errorf("panic: %v", r)
						}
						wg.Done()
					}()
					err = txmgr.monitor(mtx, ctx)
				}()

				if err != nil {
					switch err {
					// statedb errors
					case ErrRemoveMonitoredTx:
					// etherman errors
					case ErrTransactionReceipt:
					case ErrOnCanonicalChain:
					case ErrHeaderByHash:
					case ErrHeaderByNumber:
					default:
						logger.Fatal(err)
					}
				}
			}
		case <-tickerToPrepare.C:
			redeems, err := txmgr.statedb.GetRedeemByStatus(state.RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get redeems by status: err=%v", err)
				return err
			}

			for _, redeem := range redeems {
				// Check whether there is a routine currently being handling the redeem
				if _, ok := txmgr.redeemLock.Load(redeem.RequestTxHash); ok {
					continue
				}

				wg.Add(1)
				go func() {
					defer func() {
						if r := recover(); r != nil {
							txmgr.redeemLock.Delete(redeem.RequestTxHash)
						}
						wg.Done()
					}()
					err = txmgr.prepareRedeem(ctx, redeem)
				}()

				if err != nil {
					switch err {
					// context errors
					case context.Canceled:
					case context.DeadlineExceeded:
					// etherman errors
					case ErrIsPrepared:
					case ErrRedeemPrepare:
					case ErrHeaderByNumber:
					// statedb errors
					case ErrGetMonitoredTxByRequestTxHash:
					case ErrGetSignatureRequestByRequestTxHash:
					case ErrInsertMonitoredTx:
					case ErrInsertSignatureRequest:
					// wallet errors
					case ErrBtcWalletRequest:
					case ErrSchnorrWalletSign:
					default:
						logger.Fatal(err)
					}
				}
			}
		}
	}
}
