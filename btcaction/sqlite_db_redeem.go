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

// Close closes the database connection
func (s *SQLiteRedeemStorage) Close() error {
	return s.db.Close()
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

func (s *SQLiteRedeemStorage) QueryByEthRequestTxId(ethRequestTxID string) (*RedeemAction, error) {
	query := `SELECT BtcHash, Sent, Mined FROM btc_action_redeem WHERE EthRequestTxID = ?`
	var btcHash string
	var sent, mined bool
	err := s.db.QueryRow(query, ethRequestTxID).Scan(&btcHash, &sent, &mined)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &RedeemAction{
		EthRequestTxID: ethRequestTxID,
		BtcHash:        btcHash,
		Sent:           sent,
		Mined:          mined,
	}, err
}

func (s *SQLiteRedeemStorage) InsertRedeem(redeem *RedeemAction) error {
	query := `INSERT INTO btc_action_redeem (EthRequestTxID, BtcHash, Sent) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, redeem.EthRequestTxID, redeem.BtcHash, true)
	return err
}

func (s *SQLiteRedeemStorage) QueryByBtcTxId(btcTxID string) (*RedeemAction, error) {
	query := `SELECT EthRequestTxID, Sent, Mined FROM btc_action_redeem WHERE BtcHash = ?`
	redeem := &RedeemAction{}
	err := s.db.QueryRow(query, btcTxID).Scan(&redeem.EthRequestTxID, &redeem.Sent, &redeem.Mined)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	redeem.BtcHash = btcTxID
	return redeem, err
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
