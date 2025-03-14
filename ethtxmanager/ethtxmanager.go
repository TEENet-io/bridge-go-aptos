package ethtxmanager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrDBOpGetMonitoredTxs    = errors.New("failed to get monitored txs")
	ErrDBOpGetRedeemsByStatus = errors.New("failed to get redeems by status")
	ErrDBOpGetUnMinted        = errors.New("failed to get unminted")
)

type EthTxManager struct {
	cfg           *EthTxMgrConfig
	etherman      *etherman.Etherman
	statedb       *state.StateDB
	mgrdb         *EthTxManagerDB
	schnorrWallet SchnorrAsyncWallet
	btcWallet     BtcWallet

	// public key of the schnorr threshold signature
	pubKey ethcommon.Hash

	redeemLock sync.Map
	mintLock   sync.Map
}

func NewEthTxManager(
	cfg *EthTxMgrConfig,
	etherman *etherman.Etherman,
	statedb *state.StateDB,
	mgrdb *EthTxManagerDB,
	schnorrWallet SchnorrAsyncWallet,
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
	logger.Debug("starting eth tx manager")
	defer logger.Debug("stopping eth tx manager")

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

		// Read State DB & get redeems with status "requested".
		// Prepare those redeems: gather UTXOs, sign, call/post tx of "RedeemPrepare" on ETH side.
		case <-tickerToPrepare.C:
			redeemsFromDB, err := txmgr.statedb.GetRedeemsByStatus(state.RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get redeems by status: err=%v", err)
				errCh <- ErrDBOpGetRedeemsByStatus
			}

			if len(redeemsFromDB) > 0 {
				for _, redeem := range redeemsFromDB {
					logger.WithFields(logger.Fields{
						"status":    redeem.Status,
						"reqTxHash": redeem.RequestTxHash.String(),
					}).Debug("redeems (requested) from db")
				}
			} else {
				logger.WithField("num", 0).Debug("redeems=requested (db)")
			}

			redeems := []*state.Redeem{}
			for _, redeem := range redeemsFromDB {
				// Check whether there is any pending tx that has tried to prepare the redeem
				mts, err := txmgr.mgrdb.GetMonitoredTxsById(redeem.RequestTxHash)
				if err != nil {
					logger.Errorf("failed to get monitored tx in db by request tx hash: err=%v", err)
					errCh <- ErrDBOpGetMonitoredTxByID
				}
				if len(mts) == 0 { // no pending tx that has tried to prepare the redeem
					// Add the redeem to the list of redeems to be prepared
					// if there is no pending tx that has tried to prepare the redeem
					redeems = append(redeems, redeem)
				} else { // if monitored tx, but the prepre tx is timeout, re-prepare
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

			logger.WithField("num", len(redeems)).Debug("redeems (after filter) to be prepared")

			if len(redeems) == 0 {
				continue
			}

			wg := sync.WaitGroup{}
			wg.Add(len(redeems))
			for _, redeem := range redeems {
				// Check whether there is a routine currently handling the redeem
				if _, ok := txmgr.redeemLock.Load(redeem.RequestTxHash); ok {
					continue
				}

				go func() {
					defer wg.Done()
					if _prepareTxHash, err := txmgr.prepareRedeem(ctx, redeem); err != nil {
						logger.WithField("prepareTxHash", _prepareTxHash).Errorf("Error occured during prepare a Redeem: %v", err)
						// errCh <- err
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
				// Check whether there is a routine currently <handling> the mint
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
