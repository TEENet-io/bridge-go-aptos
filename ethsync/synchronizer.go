package ethsync

import (
	"context"
	"math/big"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	cfg *Config

	etherman *etherman.Etherman

	st State

	ethFinalizedBlockNumber *big.Int
	btcFinalizedBlockNumber *big.Int
}

func New(
	etherman *etherman.Etherman,
	st State,
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

	ethStored, err := st.GetEthFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	btcStored, err := st.GetBtcFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get btc finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	return &Synchronizer{
		etherman:                etherman,
		ethFinalizedBlockNumber: ethStored,
		btcFinalizedBlockNumber: btcStored,
		st:                      st,
		cfg:                     cfg,
	}, nil
}

func (s *Synchronizer) Sync(ctx context.Context) error {
	logger.Info("starting Eth synchronization")
	defer func() {
		logger.Info("stopping Eth synchronization")
	}()

	ethTicker := time.NewTicker(s.cfg.FrequencyToCheckEthFinalizedBlock)
	defer ethTicker.Stop()
	btcTicker := time.NewTicker(s.cfg.FrequencyToCheckBtcFinalizedBlock)
	defer btcTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-btcTicker.C:
			// TODO: implement
		case <-ethTicker.C:
			newFinalized, err := s.etherman.GetLatestFinalizedBlockNumber()
			if err != nil {
				return err
			}

			// continue if new finalized block number is less than the last processed block number
			if newFinalized.Cmp(s.ethFinalizedBlockNumber) != 1 {
				continue
			}

			s.st.GetNewEthFinalizedBlockChannel() <- newFinalized

			// For each block with height starting from lastFinalized + 1 to newFinalized,
			// extract all the TWBTC minted, redeem request and redeem prepared events.
			// Send all the events to the relevant states via channels.
			num := new(big.Int).Add(s.ethFinalizedBlockNumber, big.NewInt(1))
			for num.Cmp(newFinalized) != 1 {
				minted, requested, prepared, err := s.etherman.GetEventLogs(num)
				if err != nil {
					return err
				}

				for _, ev := range minted {
					logger.Debugf("found event Minted: mintTx=0x%x, amount=%v, receiver=%s", ev.TxHash, ev.Amount, ev.Receiver)
					s.st.GetNewMintedEventChannel() <- &MintedEvent{
						MintTxHash: ev.TxHash,
						BtcTxId:    ev.BtcTxId,
						Amount:     new(big.Int).Set(ev.Amount),
						Receiver:   ev.Receiver,
					}
				}

				for _, ev := range requested {
					logger.Debugf("found event RedeemRequested: reqTx=0x%x, amount=%v", ev.TxHash, ev.Amount)
					s.st.GetNewRedeemRequestedEventChannel() <- &RedeemRequestedEvent{
						RequestTxHash:   ev.TxHash,
						Requester:       ev.Sender,
						Amount:          new(big.Int).Set(ev.Amount),
						Receiver:        ev.Receiver,
						IsValidReceiver: common.IsValidBtcAddress(ev.Receiver, s.cfg.BtcChainConfig),
					}
				}

				for _, ev := range prepared {
					logger.Debugf("found event RedeemPrepared: prepTx=0x%x, reqTx=0x%x, requester=%s",
						ev.TxHash, ev.EthTxHash, ev.Requester)

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

			s.ethFinalizedBlockNumber = new(big.Int).Set(newFinalized)
		}
	}
}
