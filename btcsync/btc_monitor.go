/*
Package btcsync syncs with BTC blockchain and publishes actions to observers.
*/
package btcsync

/*
BTC monitor is a type of publisher.
It scan the btc chain and watch for interested actions.
1) deposit
2) other transfer to us

Once an interested action is found, the monitor will notify all the observers.
*/

import (
	"fmt"
	"time"

	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	myutils "github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// expose functions to let caller to register various observers before loop start.

// Loop and scrap blockchain to find interested actions.

// Once an interested action is found, notify all the observers.

const (
	CONSIDER_FINALIZED = 0               // 1 blocks old we consider finalized
	SCAN_INTERVAL      = 3 * time.Second // 3 seconds, then we scan again
)

type BTCMonitor struct {
	BridgeBTCAddress      btcutil.Address  // btc address of the bridge wallet.
	LastVistedBlockHeight int64            // last btc block height visited
	ChainConfig           *chaincfg.Params // which btc chain
	Publisher             *PublisherService
	RpcClient             *rpc.RpcClient                // rpc client to interact with btc node
	mgrState              btcaction.RedeemActionStorage // tracker of redeems.
}

// Given a BTC transaction ID, finds a record in the database
func (m *BTCMonitor) QueryRedeemTxFromDB(btcTxID string) bool {
	record, err := m.mgrState.QueryByBtcTxId(btcTxID)
	if err != nil {
		return false
	}
	if record == nil {
		return false
	}
	if len(record.EthRequestTxID) == 0 {
		return false
	}
	return true
}

// FinishRedeem marks a redeem as completed in the database
func (m *BTCMonitor) FinishRedeem(btcTxID string) string {
	record, _ := m.mgrState.QueryByBtcTxId(btcTxID)
	logger.WithField("reqTxHash", record.EthRequestTxID).Debug("Complete Redeem Action Triggered")
	_ = m.mgrState.CompleteRedeem(record.EthRequestTxID)

	record, _ = m.mgrState.QueryByEthRequestTxId(record.EthRequestTxID)
	logger.WithField("record", record).Debug("Redeem Action Record")

	return record.EthRequestTxID
}

func NewBTCMonitor(addressStr string, chainConfig *chaincfg.Params, rpcClient *rpc.RpcClient, startBlock int64, mgrState btcaction.RedeemActionStorage) (*BTCMonitor, error) {
	_address, err := btcutil.DecodeAddress(addressStr, chainConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %v", err)
	}
	return &BTCMonitor{
		BridgeBTCAddress:      _address,
		LastVistedBlockHeight: startBlock,
		ChainConfig:           chainConfig,
		Publisher:             NewPublisherService(),
		RpcClient:             rpcClient,
		mgrState:              mgrState,
	}, nil
}

// Scan represents a signle round of scanning the blockchain
// Scan for blocks,
// Scan each block for txs.
// Scan each tx for related Deposit/Transfer/Redeem actions.
// It will return nothing if success, otherwise an error
func (m *BTCMonitor) Scan() error {
	// Scrap blockchain

	// Fetch and compare lateset blocks with local records
	latestBlockHeight, err := m.RpcClient.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get latest block height: %v", err)
	}

	// If no new blocks to scan.
	if latestBlockHeight <= m.LastVistedBlockHeight {
		return nil // no blocks to scan. and no error
	}

	numbersToFetch := latestBlockHeight - m.LastVistedBlockHeight - CONSIDER_FINALIZED

	logger.WithFields(logger.Fields{
		"latestBlockHeight":     latestBlockHeight,
		"LastVistedBlockHeight": m.LastVistedBlockHeight,
		"CONSIDER_FINALIZED":    CONSIDER_FINALIZED,
		"numbersToFetch":        numbersToFetch,
	}).Debug("Scanning btc blocks")

	if numbersToFetch <= 0 {
		return nil // no blocks to scan. and no error
	}

	blocks, err := m.RpcClient.GetBlocks(int(numbersToFetch), CONSIDER_FINALIZED)

	for _, block := range blocks {
		// skip no transaction blocks
		if len(block.Transactions) == 0 {
			continue
		}
		for _, tx := range block.Transactions {
			blockHeight, err := m.RpcClient.GetBlockHeightByHash(btcutil.NewBlock(block).Hash())
			if err != nil {
				logger.WithFields(logger.Fields{
					"blockHash": btcutil.NewBlock(block).Hash(),
					"btcTxId":   tx.TxHash(),
				}).Warnf("failed to get block height by hash: %v", err)
				continue
			}
			// check if the BTC tx is a user's bridge deposit
			maybe_deposit := myutils.MaybeDepositTx(tx, m.BridgeBTCAddress, m.ChainConfig)
			if maybe_deposit {
				deposit, err := myutils.CraftDepositAction(tx, blockHeight, block, m.BridgeBTCAddress, m.ChainConfig)
				if err != nil {
					logger.WithFields(logger.Fields{
						"blockNum": blockHeight,
						"btcTxId":  tx.TxHash(),
					}).Warnf("failed to craft deposit action from a maybe_deposit: %v", err)
					continue
					//TODO: shall add REFUND BTC logic here if user actually mal-formed the deposit data.
				}
				logger.WithField("btcTxId", deposit.TxHash).Info("Deposit Found")

				observedUTXO := &ObservedUTXO{
					BlockNumber: blockHeight,
					BlockHash:   block.BlockHash().String(),
					TxID:        tx.TxHash().String(),
					Vout:        0, // deposit tx always has vout 0
					Amount:      deposit.DepositValue,
					PkScript:    tx.TxOut[0].PkScript,
				}

				// Notify Observers
				m.Publisher.NotifyDeposit(*deposit)
				m.Publisher.NotifyUTXO(*observedUTXO)
				// skip the rest of the conditions
				continue
			} else {
				transfers := myutils.MaybeJustTransfer(tx, m.BridgeBTCAddress, m.ChainConfig)
				if len(transfers) > 0 {
					for _, transfer := range transfers {
						logger.WithFields(logger.Fields{
							"btcTxId": tx.TxHash().String(),
							"vout":    transfer.Vout,
							"amount":  transfer.Amount,
						}).Info("Other Transfer Found")

						observedUTXO := &ObservedUTXO{
							BlockNumber: blockHeight,
							BlockHash:   block.BlockHash().String(),
							TxID:        tx.TxHash().String(),
							Vout:        int32(transfer.Vout),
							Amount:      transfer.Amount,
							PkScript:    tx.TxOut[transfer.Vout].PkScript,
						}

						// Notify Observers
						m.Publisher.NotifyUTXO(*observedUTXO)
					}
				}
			}

			// check if the BTC tx matches a bridge withdraw in our managment state.
			// if so, set the redeem state of mgr state to be minted.
			// notify observers to set the state on core shared state.
			_btc_txid := tx.TxHash().String()
			if m.QueryRedeemTxFromDB(_btc_txid) {

				logger.WithField("btcTxId", _btc_txid).Info("Sent Redeem Found on blockchain")

				reqTxHash := m.FinishRedeem(_btc_txid)

				// Notify Observers
				m.Publisher.NotifyRedeem(btcaction.RedeemAction{
					EthRequestTxID: reqTxHash,
					BtcHash:        _btc_txid,
					Sent:           true,
					Mined:          true,
				})
				// TODO: shall notify the "change" utxo as observedUTXO for redeem tx
				continue
			}
		}
	}
	if err != nil {
		return fmt.Errorf("failed to get finalized blocks: %v", err)
	}

	// update the last visited block height
	m.LastVistedBlockHeight = latestBlockHeight
	return nil
}

// ScanLoop continuously scans the blockchain for interested actions
func (m *BTCMonitor) ScanLoop() {
	for {
		err := m.Scan()
		if err != nil {
			logger.Warnf("BTC ScanLoop error: %v", err)
		}
		// Sleep for a while before the next scan
		time.Sleep(SCAN_INTERVAL)
	}
}
