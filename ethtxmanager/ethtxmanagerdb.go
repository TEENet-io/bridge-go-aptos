package ethtxmanager

import (
	"database/sql"
	"errors"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrMintedAtNotSet         = errors.New("mintedAt not set")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrMinedAtSetForPendingTx = errors.New("minedAt set for pending tx")
)

type EthTxManagerDB struct {
	stmtCache *database.StmtCache
}

func NewEthTxManagerDB(db *sql.DB) (*EthTxManagerDB, error) {
	if _, err := db.Exec(MonitoredTxTable); err != nil {
		return nil, err
	}

	return &EthTxManagerDB{
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (db *EthTxManagerDB) Close() {
	db.stmtCache.Clear()
}

func (db *EthTxManagerDB) InsertPendingMonitoredTx(mt *MonitoredTx) error {
	stmt, err := db.stmtCache.Prepare(queryInsertPendingMonitoredTx)
	if err != nil {
		return err
	}

	sqlMt := &sqlMonitoredTx{}
	if _, err := sqlMt.encode(mt); err != nil {
		return err
	}

	if _, err := stmt.Exec(
		sqlMt.TxHash,
		sqlMt.Id,
		sqlMt.SigningHash,
		sqlMt.Outpoints,
		sqlMt.Rx,
		sqlMt.S,
		sqlMt.SentAfter,
		string(Pending),
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) InsertMonitoredTx(mt *MonitoredTx) error {
	stmt, err := db.stmtCache.Prepare(queryInsertMonitoredTx)
	if err != nil {
		return err
	}

	sqlMt := &sqlMonitoredTx{}
	if _, err := sqlMt.encode(mt); err != nil {
		return err
	}

	var minedAt sql.NullString
	if mt.MinedAt != common.EmptyHash {
		minedAt.String = mt.MinedAt.String()[2:]
		minedAt.Valid = true
	} else {
		minedAt.Valid = false
	}

	if _, err := stmt.Exec(
		sqlMt.TxHash,
		sqlMt.Id,
		sqlMt.SigningHash,
		sqlMt.Outpoints,
		sqlMt.Rx,
		sqlMt.S,
		sqlMt.SentAfter,
		minedAt,
		sqlMt.Status,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetMonitoredTxByTxHash(txHash ethcommon.Hash) (*MonitoredTx, bool, error) {
	stmt, err := db.stmtCache.Prepare(queryGetMonitoredTxByTxHash)
	if err != nil {
		return nil, false, err
	}

	hashStr := txHash.String()[2:]

	var (
		mintedAt sql.NullString
		sqlMt    sqlMonitoredTx
	)

	if err := stmt.QueryRow(hashStr).Scan(
		&sqlMt.TxHash,
		&sqlMt.Id,
		&sqlMt.SigningHash,
		&sqlMt.Outpoints,
		&sqlMt.Rx,
		&sqlMt.S,
		&sqlMt.SentAfter,
		&mintedAt,
		&sqlMt.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	if mintedAt.Valid {
		sqlMt.MinedAt = mintedAt.String
	}

	if mt, err := sqlMt.decode(); err != nil {
		return nil, false, err
	} else {
		return mt, true, nil
	}
}

func (db *EthTxManagerDB) GetMonitoredTxsById(Id ethcommon.Hash) ([]*MonitoredTx, error) {
	stmt, err := db.stmtCache.Prepare(queryGetMonitoredTxsById)
	if err != nil {
		return nil, err
	}

	hashStr := Id.String()[2:]
	rows, err := stmt.Query(hashStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No rows found, return nil slice
		}
		return nil, err
	}
	defer rows.Close()

	var (
		mts      []*MonitoredTx
		mintedAt sql.NullString
	)
	for rows.Next() {
		var sqlMt sqlMonitoredTx
		if err := rows.Scan(
			&sqlMt.TxHash,
			&sqlMt.Id,
			&sqlMt.SigningHash,
			&sqlMt.Outpoints,
			&sqlMt.Rx,
			&sqlMt.S,
			&sqlMt.SentAfter,
			&mintedAt,
			&sqlMt.Status,
		); err != nil {
			return nil, err
		}

		if mintedAt.Valid {
			sqlMt.MinedAt = mintedAt.String
		}

		mt, err := sqlMt.decode()
		if err != nil {
			return nil, err
		}
		mts = append(mts, mt)
	}

	return mts, nil
}

func (db *EthTxManagerDB) GetMonitoredTxsByStatus(status MonitoredTxStatus) ([]*MonitoredTx, error) {
	stmt, err := db.stmtCache.Prepare(queryGetMonitoredTxsByStatus)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(string(status))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No rows found, return nil slice
		}
		return nil, err
	}
	defer rows.Close()

	var (
		mts      []*MonitoredTx
		mintedAt sql.NullString
	)

	for rows.Next() {
		var sqlMt sqlMonitoredTx
		if err := rows.Scan(
			&sqlMt.TxHash,
			&sqlMt.Id,
			&sqlMt.SigningHash,
			&sqlMt.Outpoints,
			&sqlMt.Rx,
			&sqlMt.S,
			&sqlMt.SentAfter,
			&mintedAt,
			&sqlMt.Status,
		); err != nil {
			return nil, err
		}

		if mintedAt.Valid {
			sqlMt.MinedAt = mintedAt.String
		}

		mt, err := sqlMt.decode()
		if err != nil {
			return nil, err
		}
		mts = append(mts, mt)
	}

	return mts, nil
}

func (db *EthTxManagerDB) UpdateMonitoredTxStatus(txHash ethcommon.Hash, status MonitoredTxStatus) error {
	stmt, err := db.stmtCache.Prepare(queryUpdateMonitoredTxStatus)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(status, txHash.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) UpdateMonitoredTxAfterMined(
	txHash ethcommon.Hash,
	minedAt ethcommon.Hash,
	status MonitoredTxStatus,
) error {
	if status != Success && status != Reverted {
		return ErrInvalidStatus
	}

	if minedAt == common.EmptyHash {
		return ErrMintedAtNotSet
	}

	stmt, err := db.stmtCache.Prepare(queryUpdateMonitoredTxAfterMined)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		minedAt.String()[2:],
		status,
		txHash.String()[2:],
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) DeleteMonitoredTxByTxHash(txHash ethcommon.Hash) error {
	stmt, err := db.stmtCache.Prepare(queryDeleteMonitoredTxByTxHash)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(txHash.String()[2:]); err != nil {
		return err
	}

	return nil
}
