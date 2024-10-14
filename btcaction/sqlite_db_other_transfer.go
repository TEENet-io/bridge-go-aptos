/*
This file is a detailed implementation of the SQLite storage for the OtherTransferAction struct.
Table is btc_action_other_transfer
*/
package btcaction

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteOtherTransferStorage struct {
	db *sql.DB
}

func NewSQLiteOtherTransferStorage(dbPath string) (*SQLiteOtherTransferStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	storage := &SQLiteOtherTransferStorage{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteOtherTransferStorage) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS btc_action_other_transfer (
		BlockNumber INTEGER,
		BlockHash TEXT,
		TxHash TEXT,
		Vout INTEGER,
		TransferValue INTEGER,
		TransferReceiver TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_txhash ON btc_action_other_transfer (TxHash);
	CREATE INDEX IF NOT EXISTS idx_transferreceiver ON btc_action_other_transfer (TransferReceiver);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteOtherTransferStorage) AddOtherTransfer(transfer OtherTransferAction) error {
	query := `INSERT INTO btc_action_other_transfer (BlockNumber, BlockHash, TxHash, Vout, TransferValue, TransferReceiver) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, transfer.BlockNumber, transfer.BlockHash, transfer.TxHash, transfer.Vout, transfer.TransferValue, transfer.TransferReceiver)
	return err
}

func (s *SQLiteOtherTransferStorage) GetOtherTransferByReceiver(receiver string) ([]OtherTransferAction, error) {
	query := `SELECT BlockNumber, BlockHash, TxHash, Vout, TransferValue, TransferReceiver FROM btc_action_other_transfer WHERE TransferReceiver = ?`
	rows, err := s.db.Query(query, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []OtherTransferAction
	for rows.Next() {
		var transfer OtherTransferAction
		if err := rows.Scan(&transfer.BlockNumber, &transfer.BlockHash, &transfer.TxHash, &transfer.Vout, &transfer.TransferValue, &transfer.TransferReceiver); err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	return transfers, nil
}
