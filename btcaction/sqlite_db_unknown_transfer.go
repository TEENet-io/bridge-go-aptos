package btcaction

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteUnknownTransferStorage struct {
	db *sql.DB
}

func NewSQLiteUnknownTransferStorage(dbPath string) (*SQLiteUnknownTransferStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	storage := &SQLiteUnknownTransferStorage{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteUnknownTransferStorage) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS btc_action_unknown_transfer (
		BlockNumber INTEGER,
		BlockHash TEXT,
		TxHash TEXT,
		Vout INTEGER,
		TransferValue INTEGER,
		TransferReceiver TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_txhash ON btc_action_unknown_transfer (TxHash);
	CREATE INDEX IF NOT EXISTS idx_transferreceiver ON btc_action_unknown_transfer (TransferReceiver);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteUnknownTransferStorage) AddUnknownTransfer(transfer UnknownTransferAction) error {
	query := `INSERT INTO btc_action_unknown_transfer (BlockNumber, BlockHash, TxHash, Vout, TransferValue, TransferReceiver) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, transfer.BlockNumber, transfer.BlockHash, transfer.TxHash, transfer.Vout, transfer.TransferValue, transfer.TransferReceiver)
	return err
}

func (s *SQLiteUnknownTransferStorage) GetUnknownTransferByReceiver(receiver string) ([]UnknownTransferAction, error) {
	query := `SELECT BlockNumber, BlockHash, TxHash, Vout, TransferValue, TransferReceiver FROM btc_action_unknown_transfer WHERE TransferReceiver = ?`
	rows, err := s.db.Query(query, receiver)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []UnknownTransferAction
	for rows.Next() {
		var transfer UnknownTransferAction
		if err := rows.Scan(&transfer.BlockNumber, &transfer.BlockHash, &transfer.TxHash, &transfer.Vout, &transfer.TransferValue, &transfer.TransferReceiver); err != nil {
			return nil, err
		}
		transfers = append(transfers, transfer)
	}
	return transfers, nil
}
