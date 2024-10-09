package btcvault

import (
	"fmt"
	"sync"
	"time"
)

const (
	TIMEOUT_DELAY int64 = 1800 // half an hour
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
func (tv *TreasureVault) AddUTXO(blockNumber int32, blockHash string, txID string, vout int32, amount int64) error {
	utxo := VaultUTXO{
		BlockNumber: blockNumber,
		BlockHash:   blockHash,
		TxID:        txID,
		Vout:        vout,
		Amount:      amount,
		Lockup:      false,
		Spent:       false,
		Timeout:     0,
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
func (tv *TreasureVault) ChooseAndLock(targetAmount int64) ([]VaultUTXO, error) {
	// protection against concurrent updates
	tv.updateMu.Lock()
	defer tv.updateMu.Unlock()

	utxos, err := tv.backend.QueryEnoughUTXOs(targetAmount)
	if err != nil {
		return nil, err
	}

	for i, utxo := range utxos {
		utxos[i].Lockup = true
		timepoint := time.Now().Unix() + TIMEOUT_DELAY
		utxos[i].Timeout = timepoint
		err := tv.backend.SetLockup(utxo.TxID, utxo.Vout, true)
		if err != nil {
			return nil, err
		}
		err = tv.backend.SetTimeout(utxo.TxID, utxo.Vout, timepoint)
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
