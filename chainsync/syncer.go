// ChainSync: defines common action that a syncer of chain would do (fetch events, update state, etc.)
// SyncWorker: defines the interface that a worker should implement.
package chainsync

import (
	"context"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	logger "github.com/sirupsen/logrus"
)

// Configuration
type ChainSyncConfig struct {
	IntervalCheckBlockchain time.Duration // interval to trigger the scan of blockchain.
	St                      agreement.StateChannel
	ForceScanBlkNum         int64 // retro scan block, tell Sync() to scan from this block, -1 to honor the value in state.
}

// ChainSync: defines common action that a syncer of chain would do (fetch events, update state, etc.)
type ChainSync struct {
	IntervalCheckBlockchain time.Duration          // interval to trigger the scan of blockchain.
	St                      agreement.StateChannel // state, the database.
	LastChecked             *big.Int               // laset checked ledger biggest number.
	SyncWorker              SyncWorker
}

func NewChainSync(cfg *ChainSyncConfig, syncWorker SyncWorker) (*ChainSync, error) {
	blkNumberStored, err := cfg.St.GetBlockchainFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	if cfg.ForceScanBlkNum != -1 {
		blkNumberStored = big.NewInt(cfg.ForceScanBlkNum)
	}

	return &ChainSync{
		IntervalCheckBlockchain: cfg.IntervalCheckBlockchain,
		St:                      cfg.St,
		LastChecked:             blkNumberStored,
		SyncWorker:              syncWorker,
	}, nil
}

// The Big Loop!
func (cs *ChainSync) Loop(ctx context.Context) error {
	// Ticker
	scanTicker := time.NewTicker(1 * time.Second)
	defer scanTicker.Stop()

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-scanTicker.C:
			// Fetch new finalized block number from rpc
			newFinalized, err := cs.SyncWorker.GetNewestLedgerFinalizedNumber()
			if err != nil {
				return err
			}

			blockchainGoesForward := newFinalized.Cmp(cs.LastChecked) == 1
			// continue if new finalized block number is less than the last processed block number
			if blockchainGoesForward {
				logger.WithFields(logger.Fields{
					"new_finalized_chain":  newFinalized.Int64(),
					"last_finalized_chain": cs.LastChecked.Int64(),
					"new > last?":          blockchainGoesForward,
				}).Info("Scanning blocksaaaaa")
			}

			// Blockchai doesn't go forward? skip the rest of the code.
			if !blockchainGoesForward {
				continue
			}

			// Notify the state on the new advancement of blockchain status
			cs.St.GetNewBlockChainFinalizedLedgerNumberChannel() <- newFinalized

			minted, request, prepared, err := cs.SyncWorker.GetTimeOrderedEvents(cs.LastChecked, newFinalized)
			// logger.WithField("preparedaaaaaa", prepared).Info("prepared")

			if err != nil {
				logger.WithField("error", err).Error("failed to get time ordered events")
				return err
			}
			// Notify the state!
			for _, ev := range minted {
				cs.St.GetNewMintedEventChannel() <- &ev
			}
			for _, ev := range request {
				logger.WithField("request", ev).Info("request")
				cs.St.GetNewRedeemRequestedEventChannel() <- &ev
			}
			for _, ev := range prepared {

				cs.St.GetNewRedeemPreparedEventChannel() <- &ev

			}

			cs.LastChecked = new(big.Int).Set(newFinalized)
		}
	}
}

// func (cs *ChainSync) ExecuteNewRedeemRequestedEvent(ev *agreement.RedeemRequestedEvent) error {

// 	for _, ev := range request {
// 		logger.WithField("request", ev).Info("request")
// 		cs.St.GetNewRedeemRequestedEventChannel() <- &ev
// 	}

// 	return nil
// }
