package ethsync

import (
	"context"
	"math/big"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/etherman"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	etherman *etherman.Etherman

	e2bSt Eth2BtcState
	b2eSt Btc2EthState

	// number of the last finalized block that has been processed
	lastProcessedBlockNum *big.Int

	frequencyToCheckFinalizedBlock time.Duration
}

func New(cfg *Config) *Synchronizer {
	n, err := cfg.Eth2BtcState.GetFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get finalized block number from database when initializing eth synchronizer")
		return nil
	}

	return &Synchronizer{
		etherman:                       cfg.Etherman,
		lastProcessedBlockNum:          n,
		e2bSt:                          cfg.Eth2BtcState,
		b2eSt:                          cfg.Btc2EthState,
		frequencyToCheckFinalizedBlock: cfg.FrequencyToCheckFinalizedBlock,
	}
}

func (s *Synchronizer) Sync(ctx context.Context) error {
	logger.Info("starting Eth synchronization")
	defer func() {
		logger.Info("stopping Eth synchronization")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.frequencyToCheckFinalizedBlock):
			newFinalized, err := s.etherman.GetLatestFinalizedBlockNumber()
			if err != nil {
				return err
			}

			// newFinalized <= lastFinalized
			if newFinalized.Cmp(s.lastProcessedBlockNum) != 1 {
				continue
			}

			s.e2bSt.GetNewFinalizedBlockChannel() <- newFinalized

			// For each block with height starting from lastFinalized + 1 to newFinalized,
			// extract all the TWBTC minted, redeem request and redeem prepared events.
			// Send all the events to the relevant states via channels.
			num := new(big.Int).Add(s.lastProcessedBlockNum, big.NewInt(1))
			for num.Cmp(newFinalized) != 1 {
				minted, requested, prepared, err := s.etherman.GetEventLogs(num)
				if err != nil {
					return err
				}

				for _, ev := range minted {
					s.b2eSt.GetMintedEventChannel() <- &ev
				}

				for _, ev := range requested {
					s.e2bSt.GetRedeemRequestedEventChannel() <- &ev
				}

				for _, ev := range prepared {
					s.e2bSt.GetRedeemPreparedEventChannel() <- &ev
				}

				num.Add(num, big.NewInt(1))
			}

			s.lastProcessedBlockNum = new(big.Int).Set(newFinalized)
		}
	}
}
