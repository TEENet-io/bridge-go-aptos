/*
SQLiteChainTxMgrDB implements ChainTxMgrDB.
Table is chain_tx_mgr_db

Internally,

1) If the *big.Int == nil, then it is stored as -1 in SQLite.
2) Other positive *big.Int is stored as int64 in the database.
3) If SQLite stored as -1, then when restore the object field, the field is nil.
*/
package chaintxmgrdb

import (
	"database/sql"
	"math/big"
	"strings"

	"github.com/TEENet-io/bridge-go/agreement"
	_ "github.com/mattn/go-sqlite3"
)

type SQLiteChainTxMgrDB struct {
	db *sql.DB
}

func NewSQLiteChainTxMgrDB(dbPath string) (*SQLiteChainTxMgrDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	storage := &SQLiteChainTxMgrDB{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

// Table's row structure is according to MonitoredTx
func (s *SQLiteChainTxMgrDB) init() error {
	query := `
	CREATE TABLE IF NOT EXISTS chain_tx_mgr_db (
		TxIdentifier BLOB PRIMARY KEY,
		RefIdentifier BLOB,
		SentBlockchainLedgerNumber INTEGER,
		FoundBlockchainLedgerNumber INTEGER,
		TxStatus TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_ref_identifier ON chain_tx_mgr_db (RefIdentifier);
	CREATE INDEX IF NOT EXISTS idx_tx_status ON chain_tx_mgr_db (TxStatus);
	`
	_, err := s.db.Exec(query)
	return err
}

// Implementation of the interface
func (s *SQLiteChainTxMgrDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteChainTxMgrDB) InsertMonitoredTx(tx *MonitoredTx) error {
	query := `
	INSERT INTO chain_tx_mgr_db (TxIdentifier, RefIdentifier, SentBlockchainLedgerNumber, FoundBlockchainLedgerNumber, TxStatus)
	VALUES (?, ?, ?, ?, ?);
	`
	sentLedgerNumber := int64(-1)
	if tx.SentBlockchainLedgerNumber != nil {
		sentLedgerNumber = tx.SentBlockchainLedgerNumber.Int64()
	}

	foundLedgerNumber := int64(-1)
	if tx.FoundBlockchainLedgerNumber != nil {
		foundLedgerNumber = tx.FoundBlockchainLedgerNumber.Int64()
	}

	_, err := s.db.Exec(query, tx.TxIdentifier, tx.RefIdentifier, sentLedgerNumber, foundLedgerNumber, tx.TxStatus)
	return err
}

func (s *SQLiteChainTxMgrDB) DeleteMonitoredTxByTxHash(identifier []byte) error {
	query := `
	DELETE FROM chain_tx_mgr_db WHERE TxIdentifier = ?;
	`
	_, err := s.db.Exec(query, identifier)
	return err
}

func (s *SQLiteChainTxMgrDB) GetMonitoredTxByTxIdentifier(identifier []byte) (*MonitoredTx, error) {
	query := `
	SELECT TxIdentifier, RefIdentifier, SentBlockchainLedgerNumber, FoundBlockchainLedgerNumber, TxStatus
	FROM chain_tx_mgr_db WHERE TxIdentifier = ?;
	`
	row := s.db.QueryRow(query, identifier)

	tx := &MonitoredTx{}
	var sentLedgerNumber, foundLedgerNumber int64
	err := row.Scan(&tx.TxIdentifier, &tx.RefIdentifier, &sentLedgerNumber, &foundLedgerNumber, &tx.TxStatus)
	if err == nil {
		if sentLedgerNumber == -1 {
			tx.SentBlockchainLedgerNumber = nil
		} else {
			tx.SentBlockchainLedgerNumber = big.NewInt(sentLedgerNumber)
		}
		if foundLedgerNumber == -1 {
			tx.FoundBlockchainLedgerNumber = nil
		} else {
			tx.FoundBlockchainLedgerNumber = big.NewInt(foundLedgerNumber)
		}
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return tx, err
}

func (s *SQLiteChainTxMgrDB) GetMonitoredTxByRefIdentifier(refIdentifier []byte) ([]*MonitoredTx, error) {
	query := `
	SELECT TxIdentifier, RefIdentifier, SentBlockchainLedgerNumber, FoundBlockchainLedgerNumber, TxStatus
	FROM chain_tx_mgr_db WHERE RefIdentifier = ?;
	`
	rows, err := s.db.Query(query, refIdentifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*MonitoredTx
	for rows.Next() {
		tx := &MonitoredTx{}

		var sentLedgerNumber, foundLedgerNumber int64
		if err := rows.Scan(&tx.TxIdentifier, &tx.RefIdentifier, &sentLedgerNumber, &foundLedgerNumber, &tx.TxStatus); err != nil {
			return nil, err
		}
		if sentLedgerNumber == -1 {
			tx.SentBlockchainLedgerNumber = nil
		} else {
			tx.SentBlockchainLedgerNumber = big.NewInt(sentLedgerNumber)
		}
		if foundLedgerNumber == -1 {
			tx.FoundBlockchainLedgerNumber = nil
		} else {
			tx.FoundBlockchainLedgerNumber = big.NewInt(foundLedgerNumber)
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (s *SQLiteChainTxMgrDB) GetMonitoredTxByStatus(status []agreement.MonitoredTxStatus) ([]*MonitoredTx, error) {
	query := `
	SELECT TxIdentifier, RefIdentifier, SentBlockchainLedgerNumber, FoundBlockchainLedgerNumber, TxStatus
	FROM chain_tx_mgr_db WHERE TxStatus IN (?` + strings.Repeat(", ?", len(status)-1) + `);
	`
	args := make([]interface{}, len(status))
	for i, s := range status {
		args[i] = s
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*MonitoredTx
	for rows.Next() {
		tx := &MonitoredTx{}
		var sentLedgerNumber, foundLedgerNumber int64
		if err := rows.Scan(&tx.TxIdentifier, &tx.RefIdentifier, &sentLedgerNumber, &foundLedgerNumber, &tx.TxStatus); err != nil {
			return nil, err
		}
		if sentLedgerNumber == -1 {
			tx.SentBlockchainLedgerNumber = nil
		} else {
			tx.SentBlockchainLedgerNumber = big.NewInt(sentLedgerNumber)
		}
		if foundLedgerNumber == -1 {
			tx.FoundBlockchainLedgerNumber = nil
		} else {
			tx.FoundBlockchainLedgerNumber = big.NewInt(foundLedgerNumber)
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (s *SQLiteChainTxMgrDB) UpdateRef(identifier []byte, refIdentifier []byte) error {
	query := `
	UPDATE chain_tx_mgr_db SET RefIdentifier = ? WHERE TxIdentifier = ?;
	`
	_, err := s.db.Exec(query, refIdentifier, identifier)
	return err
}

func (s *SQLiteChainTxMgrDB) UpdateSent(identifier []byte, sentAt *big.Int) error {
	query := `
	UPDATE chain_tx_mgr_db SET SentBlockchainLedgerNumber = ? WHERE TxIdentifier = ?;
	`
	_, err := s.db.Exec(query, sentAt.Int64(), identifier)
	return err
}

func (s *SQLiteChainTxMgrDB) UpdateFound(identifier []byte, foundAt *big.Int) error {
	query := `
	UPDATE chain_tx_mgr_db SET FoundBlockchainLedgerNumber = ? WHERE TxIdentifier = ?;
	`
	_, err := s.db.Exec(query, foundAt.Int64(), identifier)
	return err
}

func (s *SQLiteChainTxMgrDB) UpdateTxStatus(identifier []byte, status agreement.MonitoredTxStatus) error {
	query := `
	UPDATE chain_tx_mgr_db SET TxStatus = ? WHERE TxIdentifier = ?;
	`
	_, err := s.db.Exec(query, status, identifier)
	return err
}
