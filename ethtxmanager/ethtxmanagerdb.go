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

func (db *EthTxManagerDB) GetSignatureRequestByRequestTxHash(
	requestTxHash ethcommon.Hash,
) (*SignatureRequest, bool, error) {
	query := `SELECT * FROM signatureRequest WHERE requestTxHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	txHashStr := requestTxHash.String()[2:]

	var sqlSr sqlSignatureRequest
	if err := stmt.QueryRow(txHashStr).Scan(
		&sqlSr.RequestTxHash,
		&sqlSr.SigningHash,
		&sqlSr.Rx,
		&sqlSr.S,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	sr := &SignatureRequest{}
	return sr.restore(&sqlSr), true, nil
}

func (db *EthTxManagerDB) insertMonitoredTx(mt *monitoredTx) error {
	query := `INSERT OR IGNORE INTO monitoredTx (txHash, requestTxHash, sentAfter) VALUES (?, ?, ?)`
	stmt := db.stmtCache.MustPrepare(query)

	sqlMt := mt.covert()

	if _, err := stmt.Exec(
		sqlMt.TxHash,
		sqlMt.RequestTxHash,
		sqlMt.SentAfter,
	); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) removeMonitoredTxAfterMined(txHash ethcommon.Hash) error {
	query := `DELET FROM monitoredTx WHERE txHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	if _, err := stmt.Exec(txHash.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetMonitoredTxByRequestTxHash(
	RequestTxHash ethcommon.Hash,
) (*monitoredTx, bool, error) {
	query := `SELECT * FROM monitoredTx WHERE requestTxHash = ?`
	stmt := db.stmtCache.MustPrepare(query)

	hashStr := RequestTxHash.String()[2:]

	var sqlMt sqlmonitoredTx
	if err := stmt.QueryRow(hashStr).Scan(
		&sqlMt.TxHash,
		&sqlMt.RequestTxHash,
		&sqlMt.SentAfter,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	mt := &monitoredTx{}
	return mt.restore(&sqlMt), true, nil
}

func (db *EthTxManagerDB) GetAllMonitoredTx() ([]*monitoredTx, error) {
	query := `SELECT * FROM monitoredTx`
	stmt := db.stmtCache.MustPrepare(query)

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
		var sqlMt sqlmonitoredTx
		if err := rows.Scan(
			&sqlMt.TxHash,
			&sqlMt.RequestTxHash,
			&sqlMt.SentAfter,
		); err != nil {
			return nil, err
		}

		mt := &monitoredTx{}
		mts = append(mts, mt.restore(&sqlMt))
	}

	return mts, nil
}
