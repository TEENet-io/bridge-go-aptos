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
	BLK_MATURE_OFFSET     = 1               // ? blocks old we consider finalized
	SCAN_BTC_BLK_INTERVAL = 3 * time.Second // ? time between we scan the BTC blockchain.
	RETRO_SCAN_BLOCKS     = 36              // ? blocks to scan (to counter the BTC not producing situation).
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
func (m *BTCMonitor) QueryRedeemTxFromMgrDB(btcTxID string) bool {
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

// FinishRedeem marks a redeem as completed in the mgr database
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

	logger.WithField("btc latest blk", latestBlockHeight).Debug("Check BTC blockchain")
	logger.WithField("last visited blk", m.LastVistedBlockHeight).Debug("From memory")

	// If no new blocks to scan.
	if latestBlockHeight <= m.LastVistedBlockHeight {
		return nil // no blocks to scan. and no error
	}

	numbersToFetch := latestBlockHeight - m.LastVistedBlockHeight - BLK_MATURE_OFFSET

	// Sometimes BTC blockchain can clog,
	// for at least 6 hours (observed in testnet4),
	// we scan for at least <RETRO_SCAN_BLOCKS> blocks back.
	// like 36 blocks for 6 hours, or 72 blocks for 12 hours.
	if numbersToFetch < RETRO_SCAN_BLOCKS {
		numbersToFetch = RETRO_SCAN_BLOCKS
	}

	logger.WithFields(logger.Fields{
		"latestBlockHeight":     latestBlockHeight,
		"lastVistedBlockHeight": m.LastVistedBlockHeight,
		"considerFinalized":     BLK_MATURE_OFFSET,
		"numbersToFetch":        numbersToFetch,
	}).Info("Scanning btc blocks")

	blocks, err := m.RpcClient.GetBlocks(int(numbersToFetch), BLK_MATURE_OFFSET)

	for _, block := range blocks {
		// skip no transaction blocks
		if len(block.Transactions) == 0 {
			continue
		}

		// get block height by block_hash
		blockHeight, err := m.RpcClient.GetBlockHeightByHash(btcutil.NewBlock(block).Hash())
		if err != nil {
			logger.WithFields(logger.Fields{
				"blockHash": btcutil.NewBlock(block).Hash(),
			}).Warnf("failed to get block_height by block_hash: %v", err)
			continue
		}
		logger.WithField("blkNum", blockHeight).Info("Investigate btc block")
		// Go for each Tx, look for Tx that is interested to us.
		// In general we care about three things:
		// 1) The output(s) of the Tx, does it form a valid <bridge deposit>?
		// 2) The output(s) of the Tx, does it form a valid UTXO so we can spend in the future?
		// 3) The Tx is a redeem BTC tx that we sent?
		for _, tx := range block.Transactions {

			// 1) check if the BTC tx is a <bridge deposit>
			maybe_deposit := myutils.MaybeDepositTx(tx, m.BridgeBTCAddress, m.ChainConfig)
			if maybe_deposit {
				deposit, err := myutils.CraftDepositAction(tx, blockHeight, block, m.BridgeBTCAddress, m.ChainConfig)
				if err != nil {
					logger.WithFields(logger.Fields{
						"blockNum": blockHeight,
						"btcTxId":  tx.TxHash(),
					}).Warnf("failed to craft deposit_action from a maybe_deposit: %v", err)
					//TODO: shall add REFUND BTC logic here if user actually mal-formed the deposit data.
				} else {
					logger.WithFields(logger.Fields{
						"blockNum": blockHeight,
						"btcTxId":  deposit.TxHash,
					}).Info("Deposit Found (BTC)")
					// Notify Observers
					m.Publisher.NotifyDeposit(*deposit)
				}
			}

			// Whether or not <bridge deposit>
			// 2) We fetch ALL the UTXOs that is sending money to us (the bridge)
			transfers := myutils.MaybeJustTransfer(tx, m.BridgeBTCAddress, m.ChainConfig)
			if len(transfers) > 0 {
				for _, transfer := range transfers {
					logger.WithFields(logger.Fields{
						"blockNum": blockHeight,
						"btcTxId":  tx.TxHash().String(),
						"vout":     transfer.Vout,
						"amount":   transfer.Amount,
					}).Info("Transfer Found (BTC)")

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

			// check if the BTC tx matches a bridge withdraw in our managment state.
			// if so, set the redeem state of mgr state to be minted.
			// notify observers to set the state on core shared state.
			_btc_txid := tx.TxHash().String()
			if m.QueryRedeemTxFromMgrDB(_btc_txid) {

				logger.WithFields(logger.Fields{
					"blockNum": blockHeight,
					"btcTxId":  _btc_txid,
				}).Info("Redeem BTC Tx Found")

				reqTxHash := m.FinishRedeem(_btc_txid)

				// Notify Observers
				m.Publisher.NotifyRedeemIsDone(btcaction.RedeemAction{
					EthRequestTxID: reqTxHash,
					BtcHash:        _btc_txid,
					Sent:           true,
					Mined:          true,
				})
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
		time.Sleep(SCAN_BTC_BLK_INTERVAL)
	}
}
