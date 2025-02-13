package ethsync

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	cfg           *EthSyncConfig
	etherman      *etherman.Etherman
	st            State
	lastFinalized *big.Int
}

func New(
	etherman *etherman.Etherman,
	st State,
	cfg *EthSyncConfig,
) (*Synchronizer, error) {
	chainID, err := etherman.Client().ChainID(context.Background())
	if err != nil {
		logger.Error("failed to get eth chain ID")
		return nil, err
	}

	if chainID.Cmp(cfg.EthChainID) != 0 {
		return nil, ErrChainIDUnmatched(cfg.EthChainID, chainID)
	}

	ethStored, err := st.GetEthFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	return &Synchronizer{
		etherman:      etherman,
		lastFinalized: ethStored,
		st:            st,
		cfg:           cfg,
	}, nil
}

func (s *Synchronizer) Sync(ctx context.Context) error {
	logger.Debug("starting Eth synchronization")
	defer func() {
		logger.Debug("stopping Eth synchronization")
	}()

	ethTicker := time.NewTicker(s.cfg.FrequencyToCheckEthFinalizedBlock)
	defer ethTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ethTicker.C:
			newFinalized, err := s.etherman.GetLatestFinalizedBlockNumber()
			if err != nil {
				return err
			}

			// continue if new finalized block number is less than the last processed block number
			if newFinalized.Cmp(s.lastFinalized) != 1 {
				continue
			}

			s.st.GetNewEthFinalizedBlockChannel() <- newFinalized

			// For each block with height starting from lastFinalized + 1 to newFinalized,
			// extract all the TWBTC minted, redeem request and redeem prepared events.
			// Send all the events to the relevant states via channels.
			num := new(big.Int).Add(s.lastFinalized, big.NewInt(1))
			for num.Cmp(newFinalized) != 1 {
				minted, requested, prepared, err := s.etherman.GetEventLogs(num)
				logger.WithFields(logger.Fields{
					"minted":    len(minted),
					"requested": len(requested),
					"prepared":  len(prepared),
				}).Debug("events")
				if err != nil {
					return err
				}

				for _, ev := range minted {
					logger.WithFields(logger.Fields{
						"mintTx":        "0x" + hex.EncodeToString(ev.TxHash[:]),
						"amount":        ev.Amount,
						"receiver(btc)": ev.Receiver,
					}).Debug("Minted event")
					s.st.GetNewMintedEventChannel() <- &MintedEvent{
						MintTxHash: ev.TxHash,
						BtcTxId:    ev.BtcTxId,
						Amount:     new(big.Int).Set(ev.Amount),
						Receiver:   ev.Receiver,
					}
				}

				for _, ev := range requested {
					logger.WithFields(logger.Fields{
						"reqTx":         "0x" + hex.EncodeToString(ev.TxHash[:]),
						"amount":        ev.Amount,
						"receiver(btc)": ev.Receiver,
						"sender(evm)":   ev.Sender.String(),
					}).Debug("RedeemRequested event")
					x := &RedeemRequestedEvent{
						RequestTxHash:   ev.TxHash,
						Requester:       ev.Sender,
						Amount:          new(big.Int).Set(ev.Amount),
						Receiver:        ev.Receiver,
						IsValidReceiver: common.IsValidBtcAddress(ev.Receiver, s.cfg.BtcChainConfig),
					}
					logger.WithFields(logger.Fields{
						"requester(evm)":  x.Requester.String(),
						"receiver(btc)":   x.Receiver,
						"IsValidReceiver": x.IsValidReceiver,
					}).Debug("RedeemRequested details")
					s.st.GetNewRedeemRequestedEventChannel() <- x
				}

				for _, ev := range prepared {
					logger.WithFields(logger.Fields{
						"prepTx":         "0x" + hex.EncodeToString(ev.TxHash[:]),
						"reqTx":          "0x" + hex.EncodeToString(ev.EthTxHash[:]),
						"requester(evm)": ev.Requester.String(),
					}).Debug("RedeemPrepared event")

					outpointTxIds := []ethcommon.Hash{}
					for _, txid := range ev.OutpointTxIds {
						outpointTxIds = append(outpointTxIds, txid)
					}
					s.st.GetNewRedeemPreparedEventChannel() <- &RedeemPreparedEvent{
						PrepareTxHash: ev.TxHash,
						RequestTxHash: ev.EthTxHash,
						Requester:     ev.Requester,
						Receiver:      ev.Receiver,
						Amount:        new(big.Int).Set(ev.Amount),
						OutpointTxIds: outpointTxIds,
						OutpointIdxs:  ev.OutpointIdxs,
					}
				}

				num.Add(num, big.NewInt(1))
			}

			s.lastFinalized = new(big.Int).Set(newFinalized)
		}
	}
}
