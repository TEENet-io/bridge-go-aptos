package ethtxmanager

import (
	"database/sql"
	"errors"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/database"
)

var (
	ErrMintedAtNotSet = errors.New("mintedAt not set")
)

type EthTxManagerDB struct {
	db        *sql.DB
	stmtCache *database.StmtCache
}

func NewEthTxManagerDB(db *sql.DB) (*EthTxManagerDB, error) {
	if _, err := db.Exec(signatureRequestTable + monitoredTxTable); err != nil {
		return nil, err
	}

	return &EthTxManagerDB{
		db:        db,
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (db *EthTxManagerDB) Close() {
	db.stmtCache.Clear()
}

func (db *EthTxManagerDB) insertSignatureRequest(sr *SignatureRequest) error {
	query := `INSERT OR IGNORE INTO signatureRequest (requestTxHash, signingHash, rx, s) VALUES (?, ?, ?, ?)`
	stmt := db.stmtCache.MustPrepare(query)

	sqlSr := sr.convert()

	if _, err := stmt.Exec(
		sqlSr.RequestTxHash,
		sqlSr.SigningHash,
		sqlSr.Rx,
		sqlSr.S,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetSignatureRequestByRequestTxHash(requestTxHash [32]byte) (*SignatureRequest, error) {
	query := `SELECT * FROM signatureRequest WHERE requestTxHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	txHashStr := common.Bytes32ToHexStr(requestTxHash)[2:]

	var sqlSr sqlSignatureRequest
	if err := stmt.QueryRow(txHashStr).Scan(
		&sqlSr.RequestTxHash,
		&sqlSr.SigningHash,
		&sqlSr.Rx,
		&sqlSr.S,
	); err != nil {
		return nil, err
	}

	sr := &SignatureRequest{}
	return sr.restore(&sqlSr), nil
}

func (db *EthTxManagerDB) insertMonitoredTx(mt *monitoredTx) error {
	query := `INSERT OR IGNORE INTO monitoredTx (txHash, requestTxHash, sentAt, minedAt) VALUES (?, ?, ?, ?)`
	stmt := db.stmtCache.MustPrepare(query)

	sqlMt := mt.covert()

	if _, err := stmt.Exec(
		sqlMt.TxHash,
		sqlMt.RequestTxHash,
		sqlMt.SentAt,
		strZeroBytes32,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) updateMonitoredTxAfterMined(mt *monitoredTx) error {
	if mt.MinedAt == [32]byte{} {
		return ErrMintedAtNotSet
	}

	query := `UPDATE monitoredTx SET minedAt = ? WHERE txHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	sqlMt := mt.covert()

	if _, err := stmt.Exec(
		sqlMt.MinedAt,
		sqlMt.TxHash,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetMonitoredTxByRequestTxHash(RequestTxHash [32]byte) (*monitoredTx, error) {
	query := `SELECT * FROM monitoredTx WHERE requestTxHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	hashStr := common.Bytes32ToHexStr(RequestTxHash)[2:]

	var sqlMt sqlmonitoredTx
	if err := stmt.QueryRow(hashStr).Scan(
		&sqlMt.TxHash,
		&sqlMt.RequestTxHash,
		&sqlMt.SentAt,
		&sqlMt.MinedAt,
	); err != nil {
		return nil, err
	}

	mt := &monitoredTx{}
	return mt.restore(&sqlMt), nil
}
