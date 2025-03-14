package btcvault

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/TEENet-io/bridge-go/state"

	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
)

const (
	TIMEOUT_DELAY int64 = 3600               // an hour, then the utxo is considered released and can be used again.
	SAFE_MARGIN         = int64(0.001 * 1e8) // 0.001 BTC
)

// TreasureVault is a vault that stores UTXOs
// that is related to a btcAddress
type TreasureVault struct {
	BtcAddress string           // the wallet holds the money
	backend    VaultUTXOStorage // the backend engine
	updateMu   sync.Mutex       // prevent concurrent updates
}

// NewTreasureVault contains one btc address as identifier.
// And uses any backend that implements VaultUTXOStorage.
func NewTreasureVault(btcAddress string, backend VaultUTXOStorage) *TreasureVault {
	return &TreasureVault{BtcAddress: btcAddress, backend: backend}
}

// AddUTXO adds a new UTXO to the treasure vault
// It returns an error if the UTXO already exists (won't insert duplicates)
func (tv *TreasureVault) AddUTXO(
	blockNumber int32,
	blockHash string,
	txID string,
	vout int32,
	amount int64,
	pkScript []byte,
) error {
	utxo := VaultUTXO{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		TxID:        txID,
		Vout:        vout,
		Amount:      amount,
		PkScript:    pkScript,
		Lockup:      false,
		Spent:       false,
		Timeout:     0,
		LinkedId:    "",
	}

	// Don't duplicate insert!
	old_utxo, err := tv.backend.QueryByTxIDAndVout(txID, vout)
	if err != nil {
		return err
	}
	// Don't duplicate insert!
	if old_utxo != nil {
		return fmt.Errorf("utxo already exists")
	}

	return tv.backend.InsertVaultUTXO(utxo)
}

// ChooseAndLock selects UTXOs that sum to at least the target amount and locks them
// In bitcoin world, ususally you need to specify the target amount big enough to inlucude the fee.
// If the target amount cannot be satisfied, it will return nil + error.
// The chosen UTXOs will mark the field with linkedID (like a unique identifier for the redeem, the reqTxHash)
// It will prevent double-entry of same linkedID.
func (tv *TreasureVault) ChooseAndLock(targetAmount int64, linkedID string) ([]VaultUTXO, error) {
	// protection against concurrent updates
	tv.updateMu.Lock()
	defer tv.updateMu.Unlock()

	// Check to see if we have already locked UTXOs for the linkedID
	hits, err := tv.backend.QueryByLinkedID(linkedID)
	if err != nil {
		return nil, err
	}

	if len(hits) > 0 {
		return nil, fmt.Errorf("linkedID %s already exists, don't perform UTXO lock for it again", linkedID)
	}

	utxos, err := tv.backend.QueryEnoughUTXOs(targetAmount)
	if err != nil {
		return nil, err
	}

	for i, utxo := range utxos {
		utxos[i].Lockup = true
		err := tv.backend.SetLockup(utxo.TxID, utxo.Vout, true)
		if err != nil {
			return nil, err
		}

		timepoint := time.Now().Unix() + TIMEOUT_DELAY
		utxos[i].Timeout = timepoint
		err = tv.backend.SetTimeout(utxo.TxID, utxo.Vout, timepoint)
		if err != nil {
			return nil, err
		}

		utxos[i].LinkedId = linkedID
		err = tv.backend.SetLinkedID(utxo.TxID, utxo.Vout, linkedID)
		if err != nil {
			return nil, err
		}
	}

	return utxos, nil
}

// ReleaseByExpire releases UTXOs that have passed their timeout
func (tv *TreasureVault) ReleaseByExpire() error {
	utxos, err := tv.backend.QueryExpiredAndLockedUTXOs(time.Now().Unix())
	if err != nil {
		return err
	}

	for _, utxo := range utxos {
		err := tv.backend.SetLockup(utxo.TxID, utxo.Vout, false)
		if err != nil {
			return err
		}
		err = tv.backend.SetTimeout(utxo.TxID, utxo.Vout, 0)
		if err != nil {
			return err
		}
	}

	return nil
}

// Quick function to reveal the current state of the vault
func (tv *TreasureVault) Peek() ([]VaultUTXO, int64, error) {
	_utxos, err := tv.backend.QueryAllUTXOs()
	if err != nil {
		return nil, 0, err
	}
	_total, err := tv.backend.SumMoney()
	if err != nil {
		return nil, 0, err
	}
	return _utxos, _total, nil
}

// Vault status report
func (tv *TreasureVault) Status() {
	_utxos, sum, err := tv.Peek()

	logger.WithFields(logger.Fields{
		"sum": sum,
		"err": err,
	}).Info("UTXO Vault Status")

	if err != nil || len(_utxos) == 0 || sum == 0 {
		return // don't continue the log process.
	}

	// re-order _utxos with blocknumber from low to high
	sort.Slice(_utxos, func(i, j int) bool {
		return _utxos[i].BlockNumber < _utxos[j].BlockNumber
	})

	for _, utxo := range _utxos {
		logger.WithFields(logger.Fields{
			"txid":        utxo.TxID,
			"vout":        utxo.Vout,
			"amount":      utxo.Amount,
			"lockup":      utxo.Lockup,
			"spent":       utxo.Spent,
			"timeout":     utxo.Timeout,
			"blockNumber": utxo.BlockNumber,
			// "blockHash":   utxo.BlockHash,
		}).Info("Each UTXO Detail")
	}
}

// ReleaseByCommand releases UTXOs by their transaction ID and vout
func (tv *TreasureVault) ReleaseByCommand(txID string, vout int32) error {
	utxo, err := tv.backend.QueryByTxIDAndVout(txID, vout)
	if err != nil {
		return err
	}
	if utxo == nil {
		return fmt.Errorf("utxo not found")
	}

	err = tv.backend.SetLockup(txID, vout, false)
	if err != nil {
		return err
	}
	err = tv.backend.SetTimeout(txID, vout, 0)
	if err != nil {
		return err
	}

	return nil
}

// Fetch detailed information of a UTXO from vault.
func (tv *TreasureVault) GetUTXODetail(txID string, vout int32) (*VaultUTXO, error) {
	utxo, err := tv.backend.QueryByTxIDAndVout(txID, vout)
	if err != nil {
		return nil, err
	}
	if utxo == nil {
		return nil, fmt.Errorf("utxo not found")
	}
	return utxo, nil
}

// Implement BtcWallet interface to interact with eth_tx_manager
// eth_tx_manager will request and lock (write in smart contract)
// about the UTXOs that are collected to satisify the redeem.
// We can't collect just "barely" enough UTXOs to satisfy,
// Indeed we need to estamate the btc tx fee and add to it.
// at least leave some safe margin.
func (tv *TreasureVault) Request(
	reqTxId ethcommon.Hash,
	amount *big.Int,
	ch chan<- []state.Outpoint,
) error {
	tv.Status() // report status

	// if not enough utxos, return error
	utxos, err := tv.ChooseAndLock(amount.Int64()+SAFE_MARGIN, reqTxId.Hex())
	if err != nil {
		return err
	}

	outpoints := make([]state.Outpoint, len(utxos))
	for i, utxo := range utxos {
		outpoints[i] = state.Outpoint{
			TxId: ethcommon.HexToHash(utxo.TxID),
			Idx:  uint16(utxo.Vout),
		}
	}

	ch <- outpoints
	return nil
}
