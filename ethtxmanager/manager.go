package txmanager

import (
	"context"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
)

type TxManager struct {
	etherman    *etherman.Etherman
	toPrepareCh chan *eth2btcstate.Redeem
}

func New(etherman *etherman.Etherman) *TxManager {
	return &TxManager{
		etherman:    etherman,
		toPrepareCh: make(chan *eth2btcstate.Redeem),
	}
}

func (txmgr *TxManager) Start(ctx context.Context) error {
	logger.Info("Starting Eth transaction manager")
	defer logger.Info("Stopped Eth transaction manager")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-txmgr.toPrepareCh:
			// TODO: construct and send the transaction
		}
	}
}
