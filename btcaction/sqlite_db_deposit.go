/*
SQLiteDepositStorage represents a "storage" implemenation of "deposit action" from BTC to EVM.

It uses SQLite as the underlying storage engine.
*/
package btcaction

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	logger "github.com/sirupsen/logrus"
)

type SQLiteDepositStorage struct {
	db *sql.DB
}

func NewSQLiteDepositStorage(dbPath string) (*SQLiteDepositStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	storage := &SQLiteDepositStorage{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteDepositStorage) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS btc_action_deposit (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		block_number INTEGER,
		block_hash TEXT,
		tx_hash TEXT,
		deposit_value INTEGER,
		deposit_receiver TEXT,
		evm_id INTEGER,
		evm_addr TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_tx_hash ON btc_action_deposit(tx_hash);
	CREATE INDEX IF NOT EXISTS idx_deposit_receiver ON btc_action_deposit(deposit_receiver);
	CREATE INDEX IF NOT EXISTS idx_evm_addr ON btc_action_deposit(evm_addr);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteDepositStorage) AddDeposit(deposit DepositAction) error {
	// Protection of double adding.
	if hits, err := s.GetDepositByTxHash(deposit.TxHash); err != nil {
		return err
	} else if len(hits) > 0 {
		logger.WithField("txHash", deposit.TxHash).Debug("BTC Deposit already exists, skip.")
		return nil // no double adding.
	}
	query := `INSERT INTO btc_action_deposit (block_number, block_hash, tx_hash, deposit_value, deposit_receiver, evm_id, evm_addr) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, deposit.BlockNumber, deposit.BlockHash, deposit.TxHash, deposit.DepositValue, deposit.DepositReceiver, deposit.EvmID, deposit.EvmAddr)
	return err
}

// Fetch a list of deposit actions by btc transaction hash.
func (s *SQLiteDepositStorage) GetDepositByTxHash(txHash string) ([]DepositAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, deposit_value, deposit_receiver, evm_id, evm_addr FROM btc_action_deposit WHERE tx_hash = ?`
	rows, err := s.db.Query(query, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositAction
	for rows.Next() {
		var deposit DepositAction
		err := rows.Scan(&deposit.BlockNumber, &deposit.BlockHash, &deposit.TxHash, &deposit.DepositValue, &deposit.DepositReceiver, &deposit.EvmID, &deposit.EvmAddr)
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, deposit)
	}
	return deposits, nil
}

// Fetch a list of deposit actions by bridge address.
func (s *SQLiteDepositStorage) GetDepositByReceiver(receiver string) ([]DepositAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, deposit_value, deposit_receiver, evm_id, evm_addr FROM btc_action_deposit WHERE deposit_receiver = ?`
	rows, err := s.db.Query(query, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositAction
	for rows.Next() {
		var deposit DepositAction
		err := rows.Scan(&deposit.BlockNumber, &deposit.BlockHash, &deposit.TxHash, &deposit.DepositValue, &deposit.DepositReceiver, &deposit.EvmID, &deposit.EvmAddr)
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, deposit)
	}
	return deposits, nil
}

// Fetch a list of deposit actions by receiver EVM address and EVM ID.
func (s *SQLiteDepositStorage) GetDepositByEVM(evmAddr string, evmID int32) ([]DepositAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, deposit_value, deposit_receiver, evm_id, evm_addr FROM btc_action_deposit WHERE evm_addr = ? AND evm_id = ?`
	rows, err := s.db.Query(query, evmAddr, evmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositAction
	for rows.Next() {
		var deposit DepositAction
		err := rows.Scan(&deposit.BlockNumber, &deposit.BlockHash, &deposit.TxHash, &deposit.DepositValue, &deposit.DepositReceiver, &deposit.EvmID, &deposit.EvmAddr)
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, deposit)
	}
	return deposits, nil
}

func (s *SQLiteDepositStorage) GetDepositByEVMAddr(evmAddr string) ([]DepositAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, deposit_value, deposit_receiver, evm_id, evm_addr FROM btc_action_deposit WHERE LOWER(evm_addr) = LOWER(?)`
	rows, err := s.db.Query(query, evmAddr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deposits []DepositAction
	for rows.Next() {
		var deposit DepositAction
		err := rows.Scan(&deposit.BlockNumber, &deposit.BlockHash, &deposit.TxHash, &deposit.DepositValue, &deposit.DepositReceiver, &deposit.EvmID, &deposit.EvmAddr)
		if err != nil {
			return nil, err
		}
		deposits = append(deposits, deposit)
	}
	return deposits, nil
}
