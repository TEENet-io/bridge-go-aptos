package ethsync

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	cfg           *EthSyncConfig
	etherman      *etherman.Etherman
	st            agreement.StateChannel
	lastFinalized *big.Int
}

func New(
	etherman *etherman.Etherman,
	st agreement.StateChannel,
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

	blkNumberStored, err := st.GetEthFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get eth finalized block number from database when initializing eth synchronizer")
		return nil, err
	}

	if cfg.EthRetroScanBlkNum != -1 {
		blkNumberStored = big.NewInt(cfg.EthRetroScanBlkNum)
	}

	return &Synchronizer{
		cfg:           cfg,
		etherman:      etherman,
		st:            st,
		lastFinalized: blkNumberStored,
	}, nil
}

func (s *Synchronizer) Loop(ctx context.Context) error {
	logger.Debug("starting Eth synchronization")
	defer func() {
		logger.Debug("stopping Eth synchronization")
	}()

	ethTicker := time.NewTicker(s.cfg.IntervalCheckBlockchain)
	defer ethTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ethTicker.C:
			// Fetch new finalized block number from rpc
			newFinalized, err := s.etherman.GetLatestFinalizedBlockNumber()
			if err != nil {
				return err
			}

			newBlockFound := newFinalized.Cmp(s.lastFinalized) == 1
			// continue if new finalized block number is less than the last processed block number
			if newBlockFound {
				logger.WithFields(logger.Fields{
					"new_finalized_blk":  newFinalized.Int64(),
					"last_finalized_blk": s.lastFinalized.Int64(),
					"new > last?":        newBlockFound,
				}).Info("Scanning blocks (eth)")
			}

			// if newFinalized <= lastFinalized, skip the loop.
			if !newBlockFound {
				continue
			}

			s.st.GetNewBlockChainFinalizedLedgerNumberChannel() <- newFinalized

			// For each block with height starting from lastFinalized + 1 to newFinalized,
			// extract all the TWBTC minted, redeem request and redeem prepared events.
			// Send all the events to the relevant states via channels.
			inspecting_blk_num := new(big.Int).Add(s.lastFinalized, big.NewInt(1))
			for inspecting_blk_num.Cmp(newFinalized) != 1 {
				minted, requested, prepared, err := s.etherman.GetEventLogs(inspecting_blk_num)
				if len(minted) > 0 || len(requested) > 0 || len(prepared) > 0 {
					logger.WithFields(logger.Fields{
						"block#":    inspecting_blk_num,
						"minted":    len(minted),
						"requested": len(requested),
						"prepared":  len(prepared),
					}).Info("Inspect events from block (eth)")
				}
				if err != nil {
					return err
				}

				for _, ev := range minted {
					logger.WithFields(logger.Fields{
						"block#":        inspecting_blk_num,
						"mintTx":        "0x" + hex.EncodeToString(ev.TxHash[:]),
						"amount":        ev.Amount,
						"receiver(eth)": ev.Receiver,
					}).Info("Minted Event Found")
					s.st.GetNewMintedEventChannel() <- &agreement.MintedEvent{
						MintTxHash: ev.TxHash,
						BtcTxId:    ev.BtcTxId,
						Amount:     new(big.Int).Set(ev.Amount),
						Receiver:   ev.Receiver.Bytes(),
					}
				}

				for _, ev := range requested {
					logger.WithFields(logger.Fields{
						"block#":        inspecting_blk_num,
						"reqTx":         "0x" + hex.EncodeToString(ev.TxHash[:]),
						"amount":        ev.Amount,
						"receiver(btc)": ev.Receiver,
						"sender(evm)":   ev.Sender.String(),
					}).Info("RedeemRequested Event Found")
					x := &agreement.RedeemRequestedEvent{
						RequestTxHash:   ev.TxHash,
						Requester:       ev.Sender.Bytes(),
						Amount:          new(big.Int).Set(ev.Amount),
						Receiver:        ev.Receiver,
						IsValidReceiver: common.IsValidBtcAddress(ev.Receiver, s.cfg.BtcChainConfig),
					}
					logger.WithFields(logger.Fields{
						"block#":          inspecting_blk_num,
						"requester(evm)":  common.Prepend0xPrefix(common.ByteSliceToPureHexStr(x.Requester)),
						"receiver(btc)":   x.Receiver,
						"IsValidReceiver": x.IsValidReceiver,
					}).Debug("RedeemRequested details")
					s.st.GetNewRedeemRequestedEventChannel() <- x
				}

				for _, ev := range prepared {
					logger.WithFields(logger.Fields{
						"block#":         inspecting_blk_num,
						"prepTx":         "0x" + hex.EncodeToString(ev.TxHash[:]),
						"reqTx":          "0x" + hex.EncodeToString(ev.EthTxHash[:]),
						"requester(evm)": ev.Requester.String(),
					}).Info("RedeemPrepared Event Found")

					outpointTxIds := []ethcommon.Hash{}
					for _, txid := range ev.OutpointTxIds {
						outpointTxIds = append(outpointTxIds, txid)
					}
					s.st.GetNewRedeemPreparedEventChannel() <- &agreement.RedeemPreparedEvent{
						PrepareTxHash: ev.TxHash,
						RequestTxHash: ev.EthTxHash,
						Requester:     ev.Requester.Bytes(),
						Receiver:      ev.Receiver,
						Amount:        new(big.Int).Set(ev.Amount),
						OutpointTxIds: outpointTxIds,
						OutpointIdxs:  ev.OutpointIdxs,
					}
				}

				inspecting_blk_num.Add(inspecting_blk_num, big.NewInt(1))
			}

			s.lastFinalized = new(big.Int).Set(newFinalized)
		}
	}
}
