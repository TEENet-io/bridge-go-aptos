package btcvault

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStorage implements VaultUTXOStorage for SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLiteStorage
// dbFilePath is the path to the SQLite database file
func NewSQLiteStorage(dbFilePath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

// init initializes the VaultUTXO table and creates an index on tx_id
// if not existed before.
func (s *SQLiteStorage) init() error {
	query := `
    CREATE TABLE IF NOT EXISTS vault_utxo (
        block_number INTEGER,
        block_hash TEXT,
        tx_id TEXT,
        vout INTEGER,
        amount INTEGER,
		lockup BOOLEAN,
        spent BOOLEAN,
        timeout INTEGER,
        PRIMARY KEY (tx_id, vout)
    );
    CREATE INDEX IF NOT EXISTS idx_tx_id ON vault_utxo (tx_id);
    `
	_, err := s.db.Exec(query)
	return err
}

// InsertVaultUTXO inserts a new VaultUTXO into the database
func (s *SQLiteStorage) InsertVaultUTXO(utxo VaultUTXO) error {
	query := `
    INSERT INTO vault_utxo (block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?);
    `
	_, err := s.db.Exec(query, utxo.BlockNumber, utxo.BlockHash, utxo.TxID, utxo.Vout, utxo.Amount, utxo.Lockup, utxo.Spent, utxo.Timeout)
	return err
}

// QueryByBlockNumber retrieves all VaultUTXOs with the specified block number
func (s *SQLiteStorage) QueryByBlockNumber(blockNumber int32) ([]VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE block_number = ?;
    `
	rows, err := s.db.Query(query, blockNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.Lockup, &utxo.Spent, &utxo.Timeout); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByBlockHash retrieves all VaultUTXOs with the specified block hash
func (s *SQLiteStorage) QueryByBlockHash(blockHash string) ([]VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE block_hash = ?;
    `
	rows, err := s.db.Query(query, blockHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.Lockup, &utxo.Spent, &utxo.Timeout); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByTxID retrieves all VaultUTXOs with the specified transaction ID
func (s *SQLiteStorage) QueryByTxID(txID string) ([]VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE tx_id = ?;
    `
	rows, err := s.db.Query(query, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.Lockup, &utxo.Spent, &utxo.Timeout); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByTxIDAndVout retrieves a VaultUTXO with the specified transaction ID and vout
func (s *SQLiteStorage) QueryByTxIDAndVout(txID string, vout int32) (*VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE tx_id = ? AND vout = ?;
    `
	var utxo VaultUTXO
	err := s.db.QueryRow(query, txID, vout).Scan(
		&utxo.BlockNumber,
		&utxo.BlockHash,
		&utxo.TxID,
		&utxo.Vout,
		&utxo.Amount,
		&utxo.Lockup,
		&utxo.Spent,
		&utxo.Timeout,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No matching UTXO found
	} else if err != nil {
		return nil, err // Some other error occurred
	}
	return &utxo, nil
}

// QueryExpiredAndLockedUTXOs retrieves UTXOs whose lockup status is true and have expired
// t is the unix timepoint in seconds.
// all UTXOs with timeout < t are considered as expired.
func (s *SQLiteStorage) QueryExpiredAndLockedUTXOs(t int64) ([]VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE lockup = 1 AND timeout < ?;
    `
	rows, err := s.db.Query(query, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.Lockup, &utxo.Spent, &utxo.Timeout); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryEnoughUTXOs selects enough UTXOs to cover the specified amount
func (s *SQLiteStorage) QueryEnoughUTXOs(amount int64) ([]VaultUTXO, error) {
	query := `
    SELECT block_number, block_hash, tx_id, vout, amount, lockup, spent, timeout
    FROM vault_utxo
    WHERE lockup = 0 AND spent = 0
    ORDER BY amount DESC;
    `
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	var total int64
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.Lockup, &utxo.Spent, &utxo.Timeout); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
		total += utxo.Amount
		if total >= amount {
			break
		}
	}
	if total < amount {
		return nil, sql.ErrNoRows
	}
	return utxos, nil
}

// SetLockup sets the lockup status of a VaultUTXO identified by txID and vout
func (s *SQLiteStorage) SetLockup(txID string, vout int32, lockup bool) error {
	query := `
    UPDATE vault_utxo
    SET lockup = ?
    WHERE tx_id = ? AND vout = ?;
    `
	_, err := s.db.Exec(query, lockup, txID, vout)
	return err
}

// SetSpent sets the spent status of a VaultUTXO identified by txID and vout
func (s *SQLiteStorage) SetSpent(txID string, vout int32, spent bool) error {
	query := `
    UPDATE vault_utxo
    SET spent = ?
    WHERE tx_id = ? AND vout = ?;
    `
	_, err := s.db.Exec(query, spent, txID, vout)
	return err
}

// SetTimeout sets the expiry timepoint of a VaultUTXO identified by txID and vout
func (s *SQLiteStorage) SetTimeout(txID string, vout int32, timeout int64) error {
	query := `
    UPDATE vault_utxo
    SET timeout = ?
    WHERE tx_id = ? AND vout = ?;
    `
	_, err := s.db.Exec(query, timeout, txID, vout)
	return err
}

// SumMoney calculates the total amount of all VaultUTXOs
// Only the unspent & not locked up UTXOs are counted.
func (s *SQLiteStorage) SumMoney() (int64, error) {
	query := `
    SELECT SUM(amount)
    FROM vault_utxo
	WHERE lockup = 0 AND spent = 0;
    `
	var total int64
	err := s.db.QueryRow(query).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
