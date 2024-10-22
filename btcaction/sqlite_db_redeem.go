package btcaction

/*
	SQLiteRedeemStorage implements RedeemActionStorage using SQLite.

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
		EthRequestTxID TEXT PRIMARY KEY,
		BtcHash TEXT,
		Sent BOOLEAN DEFAULT 0,
		Mined BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_ethrequesttxid ON btc_action_redeem (EthRequestTxID);
	CREATE INDEX IF NOT EXISTS idx_btchash ON btc_action_redeem (BtcHash);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteRedeemStorage) HasRedeem(ethRequestTxID string) (bool, error) {
	query := `SELECT COUNT(*) FROM btc_action_redeem WHERE EthRequestTxID = ?`
	var count int
	err := s.db.QueryRow(query, ethRequestTxID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteRedeemStorage) InsertRedeem(redeem RedeemAction) error {
	query := `INSERT INTO btc_action_redeem (EthRequestTxID, BtcHash, Sent) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, redeem.EthRequestTxID, redeem.BtcHash, true)
	return err
}

func (s *SQLiteRedeemStorage) QueryByBtcTxId(btcTxID string) (string, error) {
	query := `SELECT EthRequestTxID FROM btc_action_redeem WHERE BtcHash = ?`
	var ethRequestTxID string // zero value of a string is ""
	err := s.db.QueryRow(query, btcTxID).Scan(&ethRequestTxID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return ethRequestTxID, err
}

func (s *SQLiteRedeemStorage) IfNotMined(ethRequestTxID string) (bool, error) {
	query := `SELECT COUNT(*) FROM btc_action_redeem WHERE EthRequestTxID = ? AND Mined = ?`
	var count int
	err := s.db.QueryRow(query, ethRequestTxID, false).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteRedeemStorage) CompleteRedeem(ethRequestTxID string) error {
	bingo, err := s.IfNotMined(ethRequestTxID)
	if err != nil {
		return err
	}
	if bingo {
		query := `UPDATE btc_action_redeem SET Mined = ? WHERE EthRequestTxID = ?`
		_, err = s.db.Exec(query, true, ethRequestTxID)
		return err
	}
	return nil
}
