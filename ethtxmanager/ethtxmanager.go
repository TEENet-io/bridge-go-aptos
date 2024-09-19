package ethtxmanager

import (
	"context"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
)

type EthTxManager struct {
	cfg *Config

	etherman *etherman.Etherman

	statedb *eth2btcstate.StateDB
	mgrdb   *EthTxManagerDB

	newSigRequestCh chan<- *eth2btcstate.Redeem
}

func New(
	cfg *Config,
	etherman *etherman.Etherman,
	statedb *eth2btcstate.StateDB,
	mgrdb *EthTxManagerDB,
	newSigRequestCh chan<- *eth2btcstate.Redeem,
) *EthTxManager {
	return &EthTxManager{
		etherman:        etherman,
		statedb:         statedb,
		cfg:             cfg,
		mgrdb:           mgrdb,
		newSigRequestCh: newSigRequestCh,
	}
}

func (txmgr *EthTxManager) Start(ctx context.Context) error {
	logger.Info("Starting eth tx manager")
	defer logger.Info("Stopped eth tx manager")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(txmgr.cfg.FrequencyToGetRequestedRedeems):
			redeems, err := txmgr.statedb.GetByStatus(eth2btcstate.RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get requested redeems from state: err=%v", err)
				return err
			}

			for _, redeem := range redeems {
				ok, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
				if err != nil {
					logger.Errorf("failed to check if redeem is prepared: tx=0x%x, err=%v", redeem.RequestTxHash, err)
					return err
				}

				if ok {
					continue
				}

				txmgr.newSigRequestCh <- redeem
			}
		}
	}
}
