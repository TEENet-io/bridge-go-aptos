package btcaction

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteWithdrawStorage struct {
	db *sql.DB
}

func NewSQLiteWithdrawStorage(dbPath string) (*SQLiteWithdrawStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create table if not exists
	query := `
	CREATE TABLE IF NOT EXISTS btc_action_withdraw (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		block_number INTEGER,
		block_hash TEXT,
		tx_hash TEXT,
		withdraw_value INTEGER,
		withdraw_receiver TEXT,
		change_value INTEGER,
		change_receiver TEXT
	);
	CREATE INDEX idx_withdraw_receiver ON btc_action_withdraw(withdraw_receiver);
	CREATE INDEX idx_tx_hash ON btc_action_withdraw(tx_hash);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &SQLiteWithdrawStorage{db: db}, nil
}

func (s *SQLiteWithdrawStorage) AddWithdraw(withdraw WithdrawAction) error {
	query := `
	INSERT INTO btc_action_withdraw (
		block_number, block_hash, tx_hash, withdraw_value, withdraw_receiver, change_value, change_receiver
	) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, withdraw.BlockNumber, withdraw.BlockHash, withdraw.TxHash, withdraw.WithdrawValue, withdraw.WithdrawReceiver, withdraw.ChangeValue, withdraw.ChangeReceiver)
	return err
}

func (s *SQLiteWithdrawStorage) GetWithdrawByTxHash(txHash string) ([]WithdrawAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, withdraw_value, withdraw_receiver, change_value, change_receiver FROM btc_action_withdraw WHERE tx_hash = ?`
	rows, err := s.db.Query(query, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []WithdrawAction
	for rows.Next() {
		var action WithdrawAction
		err := rows.Scan(&action.BlockNumber, &action.BlockHash, &action.TxHash, &action.WithdrawValue, &action.WithdrawReceiver, &action.ChangeValue, &action.ChangeReceiver)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *SQLiteWithdrawStorage) GetWithdrawByValue(value int64) ([]WithdrawAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, withdraw_value, withdraw_receiver, change_value, change_receiver FROM btc_action_withdraw WHERE withdraw_value = ?`
	rows, err := s.db.Query(query, value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []WithdrawAction
	for rows.Next() {
		var action WithdrawAction
		err := rows.Scan(&action.BlockNumber, &action.BlockHash, &action.TxHash, &action.WithdrawValue, &action.WithdrawReceiver, &action.ChangeValue, &action.ChangeReceiver)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *SQLiteWithdrawStorage) GetWithdrawByReceiver(receiver string) ([]WithdrawAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, withdraw_value, withdraw_receiver, change_value, change_receiver FROM btc_action_withdraw WHERE withdraw_receiver = ?`
	rows, err := s.db.Query(query, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []WithdrawAction
	for rows.Next() {
		var action WithdrawAction
		err := rows.Scan(&action.BlockNumber, &action.BlockHash, &action.TxHash, &action.WithdrawValue, &action.WithdrawReceiver, &action.ChangeValue, &action.ChangeReceiver)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func (s *SQLiteWithdrawStorage) GetWithdrawByChangeReceiver(receiver string) ([]WithdrawAction, error) {
	query := `SELECT block_number, block_hash, tx_hash, withdraw_value, withdraw_receiver, change_value, change_receiver FROM btc_action_withdraw WHERE change_receiver = ?`
	rows, err := s.db.Query(query, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []WithdrawAction
	for rows.Next() {
		var action WithdrawAction
		err := rows.Scan(&action.BlockNumber, &action.BlockHash, &action.TxHash, &action.WithdrawValue, &action.WithdrawReceiver, &action.ChangeValue, &action.ChangeReceiver)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}
