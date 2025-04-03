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
	st                      agreement.StateChannel
	ForceScanBlkNum         int64 // retro scan block, tell Sync() to scan from this block, -1 to honor the value in state.
}

// ChainSync: defines common action that a syncer of chain would do (fetch events, update state, etc.)
type ChainSync struct {
	IntervalCheckBlockchain time.Duration          // interval to trigger the scan of blockchain.
	st                      agreement.StateChannel // state, the database.
	lastChecked             *big.Int               // laset checked ledger biggest number.
	syncWorker              SyncWorker
}

func NewChainSync(cfg *ChainSyncConfig, syncWorker SyncWorker) (*ChainSync, error) {
	blkNumberStored, err := cfg.st.GetBlockchainFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	if cfg.ForceScanBlkNum != -1 {
		blkNumberStored = big.NewInt(cfg.ForceScanBlkNum)
	}

	return &ChainSync{
		IntervalCheckBlockchain: cfg.IntervalCheckBlockchain,
		st:                      cfg.st,
		lastChecked:             blkNumberStored,
		syncWorker:              syncWorker,
	}, nil
}

// The Big Loop!
func (cs *ChainSync) Loop(ctx context.Context) error {
	// Ticker
	scanTicker := time.NewTicker(cs.IntervalCheckBlockchain)
	defer scanTicker.Stop()

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case <-scanTicker.C:
			// Fetch new finalized block number from rpc
			newFinalized, err := cs.syncWorker.GetNewestLedgerFinalizedNumber()
			if err != nil {
				return err
			}

			blockchainGoesForward := newFinalized.Cmp(cs.lastChecked) == 1
			// continue if new finalized block number is less than the last processed block number
			if blockchainGoesForward {
				logger.WithFields(logger.Fields{
					"new_finalized_blk":  newFinalized.Int64(),
					"last_finalized_blk": cs.lastChecked.Int64(),
					"new > last?":        blockchainGoesForward,
				}).Info("Scanning blocks")
			}

			// Blockchai doesn't go forward? skip the rest of the code.
			if !blockchainGoesForward {
				continue
			}

			// Notify the state on the new advancement of blockchain status
			cs.st.GetNewBlockChainFinalizedLedgerNumberChannel() <- newFinalized

			minted, request, prepared, err := cs.syncWorker.GetTimeOrderedEvents(cs.lastChecked, newFinalized)

			if err != nil {
				return err
			}

			// Notify the state!
			for _, ev := range minted {
				cs.st.GetNewMintedEventChannel() <- &ev
			}

			for _, ev := range request {
				cs.st.GetNewRedeemRequestedEventChannel() <- &ev
			}

			for _, ev := range prepared {
				cs.st.GetNewRedeemPreparedEventChannel() <- &ev
			}

			cs.lastChecked = new(big.Int).Set(newFinalized)
		}
	}
}
