package ethtxmanager

import (
	"database/sql"
	"errors"

	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrMintedAtNotSet = errors.New("mintedAt not set")
	ErrInvalidStatus  = errors.New("invalid status")
)

type EthTxManagerDB struct {
	stmtCache *database.StmtCache
}

func NewEthTxManagerDB(db *sql.DB) (*EthTxManagerDB, error) {
	if _, err := db.Exec(signatureRequestTable + monitoredTxTable); err != nil {
		return nil, err
	}

	return &EthTxManagerDB{
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (db *EthTxManagerDB) Close() {
	db.stmtCache.Clear()
}

func (db *EthTxManagerDB) InsertSignatureRequest(sr *SignatureRequest) error {
	stmt, err := db.stmtCache.Prepare(queryInsertSignatureRequest)
	if err != nil {
		return err
	}

	sqlSr := &sqlSignatureRequest{}
	sqlSr.encode(sr)

	if _, err := stmt.Exec(
		sqlSr.RequestTxHash,
		sqlSr.SigningHash,
		sqlSr.Outpoints,
		sqlSr.Rx,
		sqlSr.S,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) RemoveSignatureRequest(requestTxHash ethcommon.Hash) error {
	stmt, err := db.stmtCache.Prepare(queryRemoveSignatureRequest)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(requestTxHash.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetSignatureRequestByRequestTxHash(
	requestTxHash ethcommon.Hash,
) (*SignatureRequest, bool, error) {
	stmt, err := db.stmtCache.Prepare(queryGetSignatureRequestByRequestTxHash)
	if err != nil {
		return nil, false, err
	}

	txHashStr := requestTxHash.String()[2:]

	var sqlSr sqlSignatureRequest
	if err := stmt.QueryRow(txHashStr).Scan(
		&sqlSr.RequestTxHash,
		&sqlSr.SigningHash,
		&sqlSr.Outpoints,
		&sqlSr.Rx,
		&sqlSr.S,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	sr, err := sqlSr.decode()
	if err != nil {
		return nil, false, err
	}

	return sr, true, nil
}

func (db *EthTxManagerDB) InsertMonitoredTx(mt *monitoredTx) error {
	stmt, err := db.stmtCache.Prepare(queryInsertMonitoredTx)
	if err != nil {
		return err
	}

	sqlMt := &sqlMonitoredTx{}
	sqlMt.encode(mt)

	if _, err := stmt.Exec(
		sqlMt.TxHash,
		sqlMt.Id,
		sqlMt.SentAfter,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) RemoveMonitoredTx(txHash ethcommon.Hash) error {
	stmt, err := db.stmtCache.Prepare(queryRemoveMonitoredTx)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(txHash.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetMonitoredTxById(
	Id ethcommon.Hash,
) (*monitoredTx, bool, error) {
	stmt, err := db.stmtCache.Prepare(queryGetMonitoredTxById)
	if err != nil {
		return nil, false, err
	}

	hashStr := Id.String()[2:]

	var sqlMt sqlMonitoredTx
	if err := stmt.QueryRow(hashStr).Scan(
		&sqlMt.TxHash,
		&sqlMt.Id,
		&sqlMt.SentAfter,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return sqlMt.decode(), true, nil
}

func (db *EthTxManagerDB) GetMonitoredTxs() ([]*monitoredTx, error) {
	stmt, err := db.stmtCache.Prepare(queryGetMonitoredTxs)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No rows found, return nil slice
		}
		return nil, err
	}
	defer rows.Close()

	var mts []*monitoredTx
	for rows.Next() {
		var sqlMt sqlMonitoredTx
		if err := rows.Scan(
			&sqlMt.TxHash,
			&sqlMt.Id,
			&sqlMt.SentAfter,
		); err != nil {
			return nil, err
		}

		mts = append(mts, sqlMt.decode())
	}

	return mts, nil
}
