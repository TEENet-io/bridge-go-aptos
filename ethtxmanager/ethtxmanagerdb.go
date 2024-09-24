package ethtxmanager

import (
	"database/sql"
	"errors"
	"fmt"

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

func (db *EthTxManagerDB) insertSignatureRequest(sr *SignatureRequest) error {
	query := `INSERT OR IGNORE INTO signatureRequest (requestTxHash, signingHash, rx, s) VALUES (?, ?, ?, ?)`
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	sqlSr := &sqlSignatureRequest{}
	sqlSr.encode(sr)

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
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

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

	return sqlSr.decode(), true, nil
}

func (db *EthTxManagerDB) insertMonitoredTx(mt *monitoredTx) error {
	query := `INSERT OR IGNORE INTO monitoredTx (txHash, requestTxHash, sentAfter) VALUES (?, ?, ?)`
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	sqlMt := &sqlMonitoredTx{}
	sqlMt.encode(mt)

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
	query := `DELETE FROM monitoredTx WHERE txHash = ?`
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(txHash.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *EthTxManagerDB) GetMonitoredTxByRequestTxHash(
	RequestTxHash ethcommon.Hash,
) (*monitoredTx, bool, error) {
	query := `SELECT * FROM monitoredTx WHERE requestTxHash = ?`
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

	hashStr := RequestTxHash.String()[2:]

	var sqlMt sqlMonitoredTx
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

	return sqlMt.decode(), true, nil
}

func (db *EthTxManagerDB) GetAllMonitoredTx() ([]*monitoredTx, error) {
	query := `SELECT * FROM monitoredTx`
	stmt, err := db.stmtCache.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query()
	if err != nil {
		if err == sql.ErrNoRows {
			return []*monitoredTx{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var mts []*monitoredTx
	for rows.Next() {
		var sqlMt sqlMonitoredTx
		if err := rows.Scan(
			&sqlMt.TxHash,
			&sqlMt.RequestTxHash,
			&sqlMt.SentAfter,
		); err != nil {
			return nil, err
		}

		mts = append(mts, sqlMt.decode())

		if mts[len(mts)-1].TxHash == (ethcommon.Hash{}) {
			fmt.Println("empty txHash")
		}
	}

	return mts, nil
}
