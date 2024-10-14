/*
BTC monitor is a type of publisher.
It scan the btc chain and watch for interested actions.
1) deposit
2) other transfer to us

Once an interested action is found, the monitor will notify all the observers.
*/
package btcsync

import (
	"fmt"
	"time"

	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

// expose functions to let caller to register various observers before loop start.

// Loop and scrap blockchain to find interested actions.

// Once an interested action is found, notify all the observers.

const (
	CONSIDER_FINALIZED = 6                // 6 blocks we consider finalized
	SCAN_INTERVAL      = 10 * time.Second // 10 seconds, then we scan again
)

type BTCMonitor struct {
	BridgeBTCAddress      btcutil.Address  // btc address of the bridge wallet.
	LastVistedBlockHeight int64            // last btc block height visited
	ChainConfig           *chaincfg.Params // which btc chain
	Publisher             *PublisherService
	RpcClient             *rpc.RpcClient // rpc client to interact with btc node
}

func NewBTCMonitor(addressStr string, chainConfig *chaincfg.Params, rpcClient *rpc.RpcClient) (*BTCMonitor, error) {
	_address, err := btcutil.DecodeAddress(addressStr, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address: %v", err)
	}
	return &BTCMonitor{
		BridgeBTCAddress:      _address,
		LastVistedBlockHeight: 0,
		ChainConfig:           chainConfig,
		Publisher:             NewPublisherService(),
		RpcClient:             rpcClient,
	}, nil
}

// Scan represents a signle round of scanning the blockchain
// It will return nothing if success, otherwise an error
func (m *BTCMonitor) Scan() error {
	// loop and scrap blockchain

	// Fetch and compare lateset blocks with local records
	latestBlockHeight, err := m.RpcClient.GetLatestBlockHeight()
	if err != nil {
		return fmt.Errorf("failed to get latest block height: %v", err)
	}

	if latestBlockHeight <= m.LastVistedBlockHeight {
		return nil // no blocks to scan. and no error
	}

	numbersToFetch := latestBlockHeight - m.LastVistedBlockHeight
	blocks, err := m.RpcClient.GetFinalizedBlocks(int(numbersToFetch), CONSIDER_FINALIZED)
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			blockHeight, err := m.RpcClient.GetBlockHeightViaHash(btcutil.NewBlock(block).Hash())
			if err != nil {
				return fmt.Errorf("failed to get block height via hash: %v", err)
			}
			// check if the tx is a deposit
			if utils.IsDepositTx(tx, m.BridgeBTCAddress, m.ChainConfig) {
				deposit, err := utils.CraftDepositAction(tx, blockHeight, block, m.BridgeBTCAddress, m.ChainConfig)
				if err != nil {
					return fmt.Errorf("failed to craft deposit action: %v", err)
					//TODO: shall add refund BTC logic here.
				}
				observedUTXO := &ObservedUTXO{
					BlockNumber: blockHeight,
					BlockHash:   block.BlockHash().String(),
					TxID:        tx.TxHash().String(),
					Vout:        0, // deposit tx always has vout 0
					Amount:      deposit.DepositValue,
				}
				m.Publisher.NotifyDeposit(*deposit)
				m.Publisher.NotifyUTXO(*observedUTXO)
			} else {
				// check if the tx is an other transfer
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
			fmt.Printf("Error during scan: %v\n", err)
		}
		// Sleep for a while before the next scan
		time.Sleep(SCAN_INTERVAL)
	}
}
