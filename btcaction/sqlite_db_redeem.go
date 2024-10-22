package btcaction

/*
SQLiteRedeemStorage is an implementation of RedeemActionStorage using SQLite.

Table is btc_action_redeem
*/

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRedeemStorage struct {
	db *sql.DB
}

func NewSQLiteRedeemStorage(dbPath string) (*SQLiteRedeemStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	storage := &SQLiteRedeemStorage{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteRedeemStorage) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS btc_action_redeem (
		BlockNumber INTEGER,
		BlockHash TEXT,
		TxHash TEXT,
		EthRequestTxID TEXT PRIMARY KEY,
		BtcHash TEXT,
		Sent BOOLEAN DEFAULT 0,
		Mined BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_ethrequesttxid ON btc_action_redeem (EthRequestTxID);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteRedeemStorage) InsertRedeem(redeem RedeemAction) error {
	query := `INSERT INTO btc_action_redeem (EthRequestTxID, BtcHash, Sent) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, redeem.EthRequestTxID, redeem.BtcHash, true)
	return err
}

func (s *SQLiteRedeemStorage) HasNotMined(ethRequestTxID string) (bool, error) {
	query := `SELECT COUNT(*) FROM btc_action_redeem WHERE EthRequestTxID = ? AND Mined = ?`
	var count int
	err := s.db.QueryRow(query, ethRequestTxID, false).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteRedeemStorage) CompleteRedeem(ethRequestTxID string, b *Basic) error {
	bingo, err := s.HasNotMined(ethRequestTxID)
	if err != nil {
		return err
	}
	if bingo {
		query := `UPDATE btc_action_redeem SET BlockNumber = ?, BlockHash = ?, TxHash = ?, Mined = ? WHERE EthRequestTxID = ?`
		_, err = s.db.Exec(query, b.BlockNumber, b.BlockHash, b.TxHash, true, ethRequestTxID)
		return err
	}
	return nil
}
