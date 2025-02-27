package btctxmanager

/*
	This file focus on EVM2BTC redeem withdraw action.

	1. Finds "prepared" redeems from local shared "state".
	2. Fetches details of UTXOs for a single given redeem.
	3. Sign and send out raw BTC Tx to do the real redeem.
	4. Insert a record in RedeemActionStorage to track the status.
*/

import (
	"fmt"
	"time"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/btcvault"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	logger "github.com/sirupsen/logrus"
)

const (
	QUERY_REDEEM_DB_INTERVAL = 10 * time.Second
	BTC_TX_FEE               = int64(0.001 * 1e8) // 0.001 BTC
)

type BtcTxManager struct {
	treasureVault *btcvault.TreasureVault       // where to query details of UTXOs.
	legacySigner  *assembler.LegacySigner       // who to sign the txs.
	myBtcClient   *rpc.RpcClient                // send/query btc blockchain.
	sharedState   *state.State                  // fetch and update the shared state. (communicate with eth side)
	mgrState      btcaction.RedeemActionStorage // tracker of redeems.
}

func NewBtcTxManager(treasureVault *btcvault.TreasureVault, legacySigner *assembler.LegacySigner, myBtcClient *rpc.RpcClient, sharedState *state.State, mgrState btcaction.RedeemActionStorage) *BtcTxManager {
	return &BtcTxManager{
		treasureVault: treasureVault,
		legacySigner:  legacySigner,
		myBtcClient:   myBtcClient,
		sharedState:   sharedState,
		mgrState:      mgrState,
	}
}

// Find "prepared" redeems from local shared "state"
func (m *BtcTxManager) FindRedeemsFromState() ([]*state.Redeem, error) {
	redeems, err := m.sharedState.GetPreparedRedeems()
	if err != nil {
		return nil, err
	}
	return redeems, nil
}

// FetchUTXOs fetches details of each UTXO for a single given redeem.
// It won't query the Bitcoin PRC, just fetch them from the local utxo vault.
func (m *BtcTxManager) FetchUTXOs(redeem *state.Redeem) ([]*btcvault.VaultUTXO, error) {
	var utxos []*btcvault.VaultUTXO
	for _, outpoint := range redeem.Outpoints {
		clean_txid := utils.Remove0xPrefix(outpoint.TxId.String())
		utxo, err := m.treasureVault.GetUTXODetail(clean_txid, int32(outpoint.Idx))
		if err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// ConvertUTXO converts a btcvault.VaultUTXO to a utxo.UTXO
// The wallet depends on utxo.UTXO type to sign transactions.
func ConvertUTXO(vaultUTXO *btcvault.VaultUTXO) *utxo.UTXO {
	txHash, _ := chainhash.NewHashFromStr(vaultUTXO.TxID)
	return &utxo.UTXO{
		TxID:      vaultUTXO.TxID,
		TxHash:    txHash,
		Vout:      uint32(vaultUTXO.Vout),
		Amount:    vaultUTXO.Amount,
		PkScriptT: utxo.ANY_SCRIPT_T,
		PkScript:  vaultUTXO.PkScript,
	}
}

// CollectUTXOs fetches UTXOs for a given redeem and converts them to utxo.UTXO type.
func (m *BtcTxManager) CollectUTXOs(redeem *state.Redeem) ([]*utxo.UTXO, error) {
	vaultUTXOs, err := m.FetchUTXOs(redeem)
	if err != nil {
		return nil, err
	}

	var utxos []*utxo.UTXO
	for _, vaultUTXO := range vaultUTXOs {
		utxos = append(utxos, ConvertUTXO(vaultUTXO))
	}

	return utxos, nil
}

// CreateBTCRedeemTx creates a redeem transaction for the given redeem.
func (m *BtcTxManager) CreateBTCRedeemTx(redeem *state.Redeem) (*wire.MsgTx, error) {
	// Collect UTXOs to be spent
	utxos, err := m.CollectUTXOs(redeem)
	if err != nil {
		return nil, err
	}

	var requestTxHash [32]byte
	copy(requestTxHash[:], redeem.RequestTxHash.Bytes())

	dst_addr := utils.Remove0xPrefix(redeem.Receiver)
	dst_amount := redeem.Amount.Int64()

	logger.WithFields(logger.Fields{
		"dst_addr":           dst_addr,
		"dst_amount":         dst_amount,
		"requestTxHash":      requestTxHash,
		"btc_tx_fee (extra)": BTC_TX_FEE,
	}).Debug("CreateBTCRedeemTx")

	redeemTx, err := m.legacySigner.MakeRedeemTx(
		dst_addr,
		dst_amount,
		requestTxHash,                        // we just fill in the eth redeem request tx hash as the identifier.
		m.legacySigner.P2PKH.EncodeAddress(), // change recevier addr
		BTC_TX_FEE,                           // TODO: remove hard code of fee.
		utxos,
	)
	if err != nil {
		return nil, err
	}

	return redeemTx, nil
}

// WithdrawBTC sends a redeem transaction to the Bitcoin network.
func (m *BtcTxManager) WithdrawBTC(redeem *state.Redeem) (*chainhash.Hash, error) {

	redeemTx, err := m.CreateBTCRedeemTx(redeem)
	if err != nil {
		return nil, err
	}

	txHash, err := m.myBtcClient.SendRawTx(redeemTx)
	if err != nil {
		return nil, err
	}

	return txHash, nil
}

// WithdrawLoop continuously finds outgoing redeems (from shared state) and processes them.
// Call it in a separate go routine.
func (m *BtcTxManager) WithdrawLoop() {
	for {
		redeems, err := m.FindRedeemsFromState()
		// if len(redeems) > 0 {
		// 	logger.Infof("Found redeems: %d", len(redeems))
		// }
		if err != nil {
			// Log the error and continue
			// Assuming there's a logger in the actual implementation
			logger.Errorf("Failed to find redeems: %v", err)
			time.Sleep(QUERY_REDEEM_DB_INTERVAL)
			continue
		}

		for _, redeem := range redeems {

			// Check if the redeem requestTxId already exists in mgrState
			reqTxHash := utils.Remove0xPrefix(redeem.RequestTxHash.String())
			exists, err := m.mgrState.HasRedeem(reqTxHash)
			if err != nil {
				// Log the error and continue with the next redeem
				logger.WithField("reqTxHash", reqTxHash).Errorf("Failed to check redeem record: %v", err)
				continue
			}

			if exists {
				ra, err := m.mgrState.QueryByEthRequestTxId(reqTxHash)
				if err != nil {
					logger.WithField("reqTxHash", reqTxHash).Errorf("Failed to query redeem record via reqTxHash: err=%v", err)
				} else {
					// If a record of the redeem already exists, continue with the next redeem
					logger.WithFields(logger.Fields{
						"reqTxHash": reqTxHash,
						"btcTxId":   ra.BtcHash,
						"sent":      ra.Sent,
						"mined":     ra.Mined,
					}).Debug("btc redeem tracked in our mgr db")
				}
				continue
			}

			// New Redeem!
			logger.WithFields(logger.Fields{
				"reqTxHash":  redeem.RequestTxHash.Hex(),
				"prepTxHash": redeem.PrepareTxHash.Hex(),
				"amount":     redeem.Amount.Int64(),
				"receiver":   redeem.Receiver,
			}).Info("new btc redeem")

			btcTxId, err := m.WithdrawBTC(redeem)
			if err != nil {
				// Log the error and continue with the next redeem
				fields := logger.Fields{
					"reqTxHash":  redeem.RequestTxHash.Hex(),
					"prepTxHash": redeem.PrepareTxHash.Hex(),
					"amount":     redeem.Amount.Int64(),
					"receiver":   redeem.Receiver,
				}
				for i, outpoint := range redeem.Outpoints {
					fields[fmt.Sprintf("outpoint_%d_txid", i)] = common.TrimHexPrefix(outpoint.TxId.Hex())
					fields[fmt.Sprintf("outpoint_%d_idx", i)] = outpoint.Idx
				}
				logger.WithFields(fields).Errorf("build & withdraw BTC tx error: %v", err)
				continue
			}
			logger.WithFields(logger.Fields{
				"reqTxHash": reqTxHash,
				"btcTxId":   btcTxId.String(),
			}).Debug("new BTC withdraw sent")

			// Insert the redeem record in mgrState (wait for future update)
			err = m.mgrState.InsertRedeem(&btcaction.RedeemAction{
				EthRequestTxID: reqTxHash,
				BtcHash:        btcTxId.String(),
				Sent:           true,
			})
			if err != nil {
				// Log the error and continue with the next redeem
				continue
			}
		}

		time.Sleep(QUERY_REDEEM_DB_INTERVAL)
	}
}
