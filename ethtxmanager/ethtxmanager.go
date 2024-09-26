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
	ErrDBOpGetUnMinted        = errors.New("failed to get unminted")
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
	mintLock   sync.Map
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
			case ErrBridgeIsMinted:
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
			case ErrDBOpGetUnMinted:
			// wallet errors
			case ErrBtcWalletRequest:
			case ErrSchnorrWalletSign:
			// statedb errors
			default:
				logger.Fatal(err)
			}
			return err
		//////////////////////////////////////////
		// TODO: handle of case of chain reorg
		// case <-reorgCh: // chain reorg detected by synchronizer
		//////////////////////////////////////////
		case <-tickerToMonitor.C:
			mtxs, err := txmgr.mgrdb.GetMonitoredTxsByStatus(Pending)
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
					if err := txmgr.monitorPendingTxs(ctx, mtx); err != nil {
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
				mts, err := txmgr.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
				if err != nil {
					logger.Errorf("failed to get monitored tx by request tx hash: err=%v", err)
					errCh <- ErrDBOpGetMonitoredTxByID
				}
				if len(mts) == 0 {
					// Add the redeem to the list of redeems to be prepared
					// if there is no pending tx that has tried to prepare the redeem
					redeems = append(redeems, redeem)
				} else {
					// Add the redeem to the list of redeems to be prepared
					// if all the txs that have tried to prepare the redeem have timed out
					isTimeout := true
					for _, mt := range mts {
						if mt.Status != Timeout {
							isTimeout = false
							break
						}
					}
					if isTimeout {
						redeems = append(redeems, redeem)
					}
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
					if err := txmgr.prepareRedeem(ctx, redeem); err != nil {
						errCh <- err
					}
				}()
			}
			wg.Wait()
		case <-tickerToMint.C:
			mintReqs, err := txmgr.statedb.GetUnMinted()
			if err != nil {
				logger.Errorf("failed to get unminted: err=%v", err)
				errCh <- ErrDBOpGetUnMinted
			}

			if len(mintReqs) == 0 {
				continue
			}

			// same rules as for check redeems applied here
			toMints := []*state.Mint{}
			for _, req := range mintReqs {
				// Check whether there is a routine currently being handling the mint
				monitoredMints, err := txmgr.mgrdb.GetMonitoredTxsById(req.BtcTxId)
				if err != nil {
					logger.Errorf("failed to get monitored tx by id: err=%v", err)
					errCh <- ErrDBOpGetMonitoredTxByID
				}
				if len(monitoredMints) == 0 {
					toMints = append(toMints, req)
				} else {
					isTimeout := true
					for _, mt := range monitoredMints {
						if mt.Status != Timeout {
							isTimeout = false
							break
						}
					}
					if isTimeout {
						toMints = append(toMints, req)
					}
				}
			}

			if len(toMints) == 0 {
				continue
			}

			wg := sync.WaitGroup{}
			wg.Add(len(toMints))
			for _, toMint := range toMints {
				if _, ok := txmgr.mintLock.Load(toMint.BtcTxId); ok {
					continue
				}

				go func() {
					defer wg.Done()
					if err := txmgr.mint(ctx, toMint); err != nil {
						errCh <- err
					}
				}()
			}
			wg.Wait()
		}
	}
}
