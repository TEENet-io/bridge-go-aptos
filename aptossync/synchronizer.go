package aptossync

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/aptosman"
	"github.com/TEENet-io/bridge-go/common"
	logger "github.com/sirupsen/logrus"
)

const MinTickerDuration = 100 * time.Millisecond

type Synchronizer struct {
	cfg           *AptosSyncConfig
	aptosman      *aptosman.Aptosman
	st            agreement.StateChannel
	lastFinalized *big.Int
}

func ErrChainIDUnmatched(expected, got string) error {
	return fmt.Errorf("chain ID unmatched: expected %s, got %s", expected, got)
}

// StringToBytes32 converts a string to a fixed-size 32-byte array
// If the input string is shorter than 32 bytes, it will be padded with zeros
// If the input string is longer than 32 bytes, it will be truncated
func StringToBytes32(s string) [32]byte {
	var result [32]byte
	b := []byte(s)

	// Copy the bytes from the input string to the result array
	// If b is shorter than 32 bytes, only that many bytes will be copied
	// If b is longer than 32 bytes, only the first 32 bytes will be copied
	copy(result[:], b)

	return result
}

// HexStringToBytes32 converts a hex string to a fixed-size 32-byte array
// The input string should be a hex representation (with or without 0x prefix)
// If the decoded bytes are shorter than 32 bytes, the result will be padded with zeros
// If the decoded bytes are longer than 32 bytes, they will be truncated
func HexStringToBytes32(hexStr string) ([32]byte, error) {
	var result [32]byte

	// Remove 0x prefix if present
	if len(hexStr) >= 2 && hexStr[0:2] == "0x" {
		hexStr = hexStr[2:]
	}

	// Decode hex string to bytes
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, fmt.Errorf("failed to decode hex string: %v", err)
	}

	// Copy the decoded bytes to the result array
	copy(result[:], b)

	return result, nil
}

func New(
	aptosman *aptosman.Aptosman,
	st agreement.StateChannel,
	cfg *AptosSyncConfig,
) (*Synchronizer, error) {
	if aptosman == nil {
		return nil, fmt.Errorf("aptosman instance is nil")
	}
	logger.WithFields(logger.Fields{
		"aptosman": fmt.Sprintf("%+v", aptosman),
		"config":   fmt.Sprintf("%+v", cfg),
	}).Debug("Creating new synchronizer")

	blkNumberStored, err := st.GetBlockchainFinalizedBlockNumber()
	if err != nil {
		logger.Error("failed to get aptos finalized version from database when initializing aptos synchronizer")
		return nil, err
	}

	if cfg.AptosRetroScanBlkNum != -1 {
		blkNumberStored = big.NewInt(cfg.AptosRetroScanBlkNum)
	}

	return &Synchronizer{
		cfg:           cfg,
		aptosman:      aptosman,
		st:            st,
		lastFinalized: blkNumberStored,
	}, nil
}

func (s *Synchronizer) Loop(ctx context.Context) error {
	defer func() {
		logger.Debug("stopping Aptos synchronization")
	}()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		fmt.Println("PASS FOR")
		select {
		case <-ctx.Done():
			fmt.Println("PASS CONTEXT DONE")
			return ctx.Err()
		case <-ticker.C:
			if s.aptosman == nil {
				logger.Error("aptosman is nil")
				return fmt.Errorf("aptosman is nil")
			}
			newFinalized, err := s.aptosman.GetLatestFinalizedVersion()
			logger.WithField("newFinalizedaaaa", newFinalized).Info("newFinalized")
			if err != nil {
				logger.WithError(err).Error("failed to get latest finalized version")
				return err
			}

			if s.lastFinalized == nil {
				logger.Error("lastFinalized is nil")
				s.lastFinalized = big.NewInt(0)
			}

			newVersionFound := newFinalized > s.lastFinalized.Uint64()
			if newVersionFound {
				logger.WithFields(logger.Fields{
					"new_finalized_version":  newFinalized,
					"last_finalized_version": s.lastFinalized,
					"new > last?":            newVersionFound,
				}).Info("Scanning versions (aptos)")
			}
			if !newVersionFound {
				continue
			}
			s.st.GetNewBlockChainFinalizedLedgerNumberChannel() <- big.NewInt(int64(newFinalized))
			inspecting_version := new(big.Int).Add(s.lastFinalized, big.NewInt(1))
			for inspecting_version.Cmp(big.NewInt(int64(newFinalized))) != 1 {
				logger.WithField("inspecting_version", inspecting_version).Info("inspecting_version")
				minted, requested, prepared, err := s.aptosman.GetModuleEvents(inspecting_version.Uint64(), newFinalized)
				logger.WithField("prepared", prepared).Info("prepared")
				if len(minted) > 0 || len(requested) > 0 || len(prepared) > 0 {
					logger.WithFields(logger.Fields{
						"version#":  inspecting_version,
						"minted":    len(minted),
						"requested": len(requested),
						"prepared":  len(prepared),
					}).Info("Inspect events from version (aptos)")
				}
				if err != nil {
					return err
				}
				for _, ev := range minted {
					logger.WithFields(logger.Fields{
						"version#":        inspecting_version,
						"mintTx":          ev.MintTxHash,
						"amount":          ev.Amount,
						"receiver(aptos)": ev.Receiver,
					}).Info("Minted Event Found")
					amount := new(big.Int).SetUint64(ev.Amount)
					s.st.GetNewMintedEventChannel() <- &agreement.MintedEvent{
						MintTxHash: common.HexStrToBytes32(ev.MintTxHash),
						BtcTxId:    common.HexStrToBytes32(ev.BtcTxId),
						Amount:     amount,
						Receiver:   []byte(ev.Receiver),
					}
				}
				for _, ev := range requested {
					logger.WithFields(logger.Fields{
						"version#":      inspecting_version,
						"reqTx":         ev.RequestTxHash,
						"amount":        ev.Amount,
						"receiver(btc)": ev.Receiver,
						"sender(aptos)": ev.Requester,
					}).Info("RedeemRequested Event Found")

					amount := new(big.Int).SetUint64(ev.Amount)

					var isValidReceiver bool
					if s.cfg.BtcChainConfig != nil {
						isValidReceiver = common.IsValidBtcAddress(ev.Receiver, s.cfg.BtcChainConfig)
					} else {
						logger.Warn("BtcChainConfig is nil, skipping address validation")
						isValidReceiver = false
					}

					x := &agreement.RedeemRequestedEvent{
						RequestTxHash:   common.HexStrToBytes32(ev.RequestTxHash),
						Requester:       []byte(ev.Requester),
						Amount:          amount,
						Receiver:        ev.Receiver,
						IsValidReceiver: isValidReceiver,
					}
					s.st.GetNewRedeemRequestedEventChannel() <- x
				}
				for _, ev := range prepared {
					logger.WithFields(logger.Fields{
						"version#":         inspecting_version,
						"prepTx":           ev.PrepareTxHash,
						"reqTx":            ev.RequestTxHash,
						"requester(aptos)": ev.Requester,
					}).Info("RedeemPrepared Event Found")
					amount := new(big.Int).SetUint64(ev.Amount)
					s.st.GetNewRedeemPreparedEventChannel() <- &agreement.RedeemPreparedEvent{
						PrepareTxHash: common.HexStrToBytes32(ev.PrepareTxHash),
						RequestTxHash: common.HexStrToBytes32(ev.RequestTxHash),
						Requester:     []byte(ev.Requester),
						Receiver:      ev.Receiver,
						Amount:        amount,
						OutpointTxIds: common.ArrayHexStrToHashes(ev.OutpointTxIds),
						OutpointIdxs:  ev.OutpointIdxs,
					}
				}
				inspecting_version.Add(inspecting_version, big.NewInt(1))
			}

			s.lastFinalized = big.NewInt(int64(newFinalized))
		}
	}
}
