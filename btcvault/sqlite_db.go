package btcvault

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// VaultSQLiteStorage implements VaultUTXOStorage for SQLite
type VaultSQLiteStorage struct {
	uniqueTableID string
	db            *sql.DB
}

// NewVaultSQLiteStorage creates a new SQLiteStorage
// dbFilePath is the path to the SQLite database file
func NewVaultSQLiteStorage(dbFilePath string, uniqueID string) (*VaultSQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}

	storage := &VaultSQLiteStorage{db: db, uniqueTableID: "vault_utxo_" + uniqueID}
	if err := storage.init(); err != nil {
		return nil, err
	}

	return storage, nil
}

// init initializes the VaultUTXO table and creates an index on tx_id
// if not existed before.
func (s *VaultSQLiteStorage) init() error {
	query := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s (
		block_number INTEGER,
		block_hash TEXT,
		tx_id TEXT,
		vout INTEGER,
		amount INTEGER,
		pkscript BLOB,
		lockup BOOLEAN,
		spent BOOLEAN,
		timeout INTEGER,
		linked_id TEXT,
		PRIMARY KEY (tx_id, vout)
	);
	CREATE INDEX IF NOT EXISTS idx_tx_id ON %s (tx_id);
	`, s.uniqueTableID, s.uniqueTableID)
	_, err := s.db.Exec(query)
	return err
}

// InsertVaultUTXO inserts a new VaultUTXO into the database
func (s *VaultSQLiteStorage) InsertVaultUTXO(utxo VaultUTXO) error {
	query := fmt.Sprintf(`
	INSERT INTO %s (block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, s.uniqueTableID)
	_, err := s.db.Exec(query, utxo.BlockNumber, utxo.BlockHash, utxo.TxID, utxo.Vout, utxo.Amount, utxo.PkScript, utxo.Lockup, utxo.Spent, utxo.Timeout, utxo.LinkedId)
	return err
}

func (s *VaultSQLiteStorage) QueryByLinkedID(linkedID string) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE linked_id = ?;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query, linkedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

func (s *VaultSQLiteStorage) QueryAllUTXOs() ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

func (s *VaultSQLiteStorage) QueryAllUsableUTXOs() ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE lockup = 0 AND spent = 0;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByBlockNumber retrieves all VaultUTXOs with the specified block number
func (s *VaultSQLiteStorage) QueryByBlockNumber(blockNumber int32) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE block_number = ?;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query, blockNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByBlockHash retrieves all VaultUTXOs with the specified block hash
func (s *VaultSQLiteStorage) QueryByBlockHash(blockHash string) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE block_hash = ?;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query, blockHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByTxID retrieves all VaultUTXOs with the specified transaction ID
func (s *VaultSQLiteStorage) QueryByTxID(txID string) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE tx_id = ?;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryByTxIDAndVout retrieves a VaultUTXO with the specified transaction ID and vout
func (s *VaultSQLiteStorage) QueryByTxIDAndVout(txID string, vout int32) (*VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE tx_id = ? AND vout = ?;
	`, s.uniqueTableID)
	var utxo VaultUTXO
	err := s.db.QueryRow(query, txID, vout).Scan(
		&utxo.BlockNumber,
		&utxo.BlockHash,
		&utxo.TxID,
		&utxo.Vout,
		&utxo.Amount,
		&utxo.PkScript,
		&utxo.Lockup,
		&utxo.Spent,
		&utxo.Timeout,
		&utxo.LinkedId,
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
func (s *VaultSQLiteStorage) QueryExpiredAndLockedUTXOs(t int64) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE lockup = 1 AND timeout < ?;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// QueryEnoughUTXOs selects enough UTXOs to cover the specified amount
// If amount cannot be satisified, will return (nil, error)
func (s *VaultSQLiteStorage) QueryEnoughUTXOs(amount int64) ([]VaultUTXO, error) {
	query := fmt.Sprintf(`
	SELECT block_number, block_hash, tx_id, vout, amount, pkscript, lockup, spent, timeout, linked_id
	FROM %s
	WHERE lockup = 0 AND spent = 0
	ORDER BY amount DESC;
	`, s.uniqueTableID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var utxos []VaultUTXO
	var total int64
	for rows.Next() {
		var utxo VaultUTXO
		if err := rows.Scan(&utxo.BlockNumber, &utxo.BlockHash, &utxo.TxID, &utxo.Vout, &utxo.Amount, &utxo.PkScript, &utxo.Lockup, &utxo.Spent, &utxo.Timeout, &utxo.LinkedId); err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
		total += utxo.Amount
		if total >= amount {
			break
		}
	}
	if total < amount {
		return nil, fmt.Errorf("not enough UTXOs to cover the amount required, required=%v, have=%v", amount, total)
	}
	return utxos, nil
}

// SetLockup sets the lockup status of a VaultUTXO identified by txID and vout
func (s *VaultSQLiteStorage) SetLockup(txID string, vout int32, lockup bool) error {
	query := fmt.Sprintf(`
	UPDATE %s
	SET lockup = ?
	WHERE tx_id = ? AND vout = ?;
	`, s.uniqueTableID)
	_, err := s.db.Exec(query, lockup, txID, vout)
	return err
}

// SetSpent sets the spent status of a VaultUTXO identified by txID and vout
func (s *VaultSQLiteStorage) SetSpent(txID string, vout int32, spent bool) error {
	query := fmt.Sprintf(`
	UPDATE %s
	SET spent = ?
	WHERE tx_id = ? AND vout = ?;
	`, s.uniqueTableID)
	_, err := s.db.Exec(query, spent, txID, vout)
	return err
}

func (s *VaultSQLiteStorage) SetLinkedID(txID string, vout int32, linkedID string) error {
	query := fmt.Sprintf(`
	UPDATE %s
	SET linked_id = ?
	WHERE tx_id = ? AND vout = ?;
	`, s.uniqueTableID)
	_, err := s.db.Exec(query, linkedID, txID, vout)
	return err
}

// SetTimeout sets the expiry timepoint of a VaultUTXO identified by txID and vout
func (s *VaultSQLiteStorage) SetTimeout(txID string, vout int32, timeout int64) error {
	query := fmt.Sprintf(`
	UPDATE %s
	SET timeout = ?
	WHERE tx_id = ? AND vout = ?;
	`, s.uniqueTableID)
	_, err := s.db.Exec(query, timeout, txID, vout)
	return err
}

// SumMoney calculates the total amount of all VaultUTXOs
// Only the unspent & not locked up UTXOs are counted.
func (s *VaultSQLiteStorage) SumMoney() (int64, error) {
	// If SUM(amount) == NULL then will return 0
	query := fmt.Sprintf(`
	SELECT COALESCE(SUM(amount), 0)
	FROM %s
	WHERE lockup = 0 AND spent = 0;
	`, s.uniqueTableID)
	var total int64
	err := s.db.QueryRow(query).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
