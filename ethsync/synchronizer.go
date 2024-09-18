package ethsync

import (
	"context"
	"math/big"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	cfg *Config

	etherman *etherman.Etherman

	e2bSt Eth2BtcState
	b2eSt Btc2EthState

	// number of the last finalized block that has been processed
	lastProcessedBlockNum *big.Int
}

func New(
	etherman *etherman.Etherman,
	e2bstate Eth2BtcState,
	b2estate Btc2EthState,
	cfg *Config,
) (*Synchronizer, error) {
	chainID, err := etherman.Client().ChainID(context.Background())
	if err != nil {
		logger.Error("failed to get eth chain ID")
		return nil, err
	}

	if chainID.Cmp(cfg.EthChainID) != 0 {
		return nil, ErrChainIDUnmatched(cfg.EthChainID, chainID)
	}

	stored, err := e2bstate.GetFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	if stored.Cmp(common.EthStartingBlock) == -1 {
		return nil, ErrStoredFinalizedBlockNumberInvalid(stored, common.EthStartingBlock)
	}

	return &Synchronizer{
		etherman:              etherman,
		lastProcessedBlockNum: stored,
		e2bSt:                 e2bstate,
		b2eSt:                 b2estate,
		cfg:                   cfg,
	}, nil
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
		case <-time.After(s.cfg.FrequencyToCheckFinalizedBlock):
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
					s.b2eSt.GetNewMintedEventChannel() <- &MintedEvent{
						MintedTxHash: ev.TxHash,
						BtcTxId:      ev.BtcTxId,
						Amount:       new(big.Int).Set(ev.Amount),
						Receiver:     ev.Receiver,
					}
				}

				for _, ev := range requested {
					s.e2bSt.GetNewRedeemRequestedEventChannel() <- &RedeemRequestedEvent{
						RedeemRequestTxHash: ev.TxHash,
						Requester:           ev.Sender,
						Amount:              new(big.Int).Set(ev.Amount),
						Receiver:            ev.Receiver,
						IsValidReceiver:     common.IsValidBtcAddress(ev.Receiver, s.cfg.BtcChainConfig),
					}
				}

				for _, ev := range prepared {
					s.e2bSt.GetNewRedeemPreparedEventChannel() <- &RedeemPreparedEvent{
						RedeemPrepareTxHash: ev.TxHash,
						RedeemRequestTxHash: ev.EthTxHash,
						Requester:           ev.Requester,
						Receiver:            ev.Receiver,
						Amount:              new(big.Int).Set(ev.Amount),
						OutpointTxIds:       ev.OutpointTxIds,
						OutpointIdxs:        ev.OutpointIdxs,
					}
				}

				num.Add(num, big.NewInt(1))
			}

			s.lastProcessedBlockNum = new(big.Int).Set(newFinalized)
		}
	}
}
