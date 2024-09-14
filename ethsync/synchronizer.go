package ethsync

import (
	"context"
	"math/big"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
)

const MinTickerInterval = 100 * time.Millisecond

type Synchronizer struct {
	etherman                     *etherman.Etherman
	lastFinalized                *big.Int
	e2bSt                        Eth2BtcState
	b2eSt                        Btc2EthState
	checkFinalizedTickerInterval time.Duration
}

func New(cfg *Config) *Synchronizer {
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

	return &Synchronizer{
		etherman:                     cfg.Etherman,
		lastFinalized:                new(big.Int).Set(cfg.LastFinalizedBLock),
		e2bSt:                        cfg.Eth2BtcState,
		b2eSt:                        cfg.Btc2EthState,
		checkFinalizedTickerInterval: d,
	}
}

func (s *Synchronizer) Sync(ctx context.Context) error {
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

			// For each block with height starting from lastFinalized + 1 to newFinalized,
			// extract all the TWBTC minted, redeem request and redeem prepared events.
			// Send all the events to the relevant states via channels.
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
