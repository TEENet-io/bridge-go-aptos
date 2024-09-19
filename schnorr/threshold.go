package schnorr

import (
	"context"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
)

type Thresholder struct {
	cfg *Config

	e2bstate *eth2btcstate.State

	newSignatureRequestCh chan *eth2btcstate.Redeem
}

func NewThresholder(cfg *Config, e2bstate *eth2btcstate.State) *Thresholder {
	return &Thresholder{
		cfg:                   cfg,
		e2bstate:              e2bstate,
		newSignatureRequestCh: make(chan *eth2btcstate.Redeem, cfg.ChannelSize),
	}
}

func (t *Thresholder) Start(ctx context.Context) error {
	logger.Info("Starting schnorr thresholder")
	defer logger.Info("Stopping schnorr thresholder")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.newSignatureRequestCh:
		}
	}
}

func (t *Thresholder) GetNewSigRequestChan() chan<- *eth2btcstate.Redeem {
	return t.newSignatureRequestCh
}
