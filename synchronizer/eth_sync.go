package synchronizer

import (
	"context"
	"math/big"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
)

const MinTickerInterval = 100 * time.Millisecond

type EthSynchronizer struct {
	etherman                     *etherman.Etherman
	lastFinalized                *big.Int
	e2bSt                        state.Eth2BtcState
	b2eSt                        state.Btc2EthState
	checkFinalizedTickerInterval time.Duration
}

func NewEthSynchronizer(cfg *EthSyncConfig) *EthSynchronizer {
	if cfg.LastFinalizedBLock.Cmp(common.EthStartingBlock) == -1 ||
		cfg.Etherman == nil ||
		cfg.Eth2BtcState == nil ||
		cfg.Btc2EthState == nil {
		return nil
	}

	var d time.Duration
	if cfg.CheckFinalizedTickerInterval <= MinTickerInterval {
		d = MinTickerInterval
	} else {
		d = cfg.CheckFinalizedTickerInterval
	}

	return &EthSynchronizer{
		etherman:                     cfg.Etherman,
		lastFinalized:                cfg.LastFinalizedBLock,
		e2bSt:                        cfg.Eth2BtcState,
		b2eSt:                        cfg.Btc2EthState,
		checkFinalizedTickerInterval: d,
	}
}

func (s *EthSynchronizer) Sync(ctx context.Context) error {
	logger.Info("starting Eth synchronization")
	ticker := time.NewTicker(s.checkFinalizedTickerInterval)
	defer func() {
		logger.Info("stopping Eth synchronization")
		ticker.Stop()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newFinalized, err := s.etherman.GetLatestFinalizedBlockNumber()
			if err != nil {
				return err
			}

			// newFinalized <= lastFinalized
			if newFinalized.Cmp(s.lastFinalized) != 1 {
				continue
			}

			s.e2bSt.GetLastEthFinalizedBlockNumberChannel() <- newFinalized

			// starting from lastFinalized + 1 to newFinalized
			blkNum := new(big.Int).Add(s.lastFinalized, big.NewInt(1))
			for blkNum.Cmp(newFinalized) != 1 {
				minted, requested, prepared, err := s.etherman.GetEventLogs(blkNum)
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

				blkNum.Add(blkNum, big.NewInt(1))
			}

			s.lastFinalized = new(big.Int).Set(newFinalized)
		}
	}
}
