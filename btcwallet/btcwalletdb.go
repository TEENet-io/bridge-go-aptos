package btcwallet

import (
	"database/sql"
	"math/big"

	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type BtcWalletDB struct {
	stmtcache *database.StmtCache
}

func NewBtcWalletDB(db *sql.DB) (*BtcWalletDB, error) {
	if _, err := db.Exec(spendableTable + sortedSpendable + requestTable); err != nil {
		return nil, err
	}

	return &BtcWalletDB{
		stmtcache: database.NewStmtCache(db),
	}, nil
}

func (db *BtcWalletDB) Close() {
	db.stmtcache.Clear()
}

func (db *BtcWalletDB) InsertSpendable(spendable *Spendable) error {
	stmt, err := db.stmtcache.Prepare(queryInsertSpendable)
	if err != nil {
		return err
	}

	var sqlSpendable sqlSpendable
	if _, err := sqlSpendable.encode(spendable); err != nil {
		return err
	}

	if _, err := stmt.Exec(
		sqlSpendable.BtcTxId,
		sqlSpendable.Idx,
		sqlSpendable.Amount,
		sqlSpendable.BlockNumber,
		sqlSpendable.Lock,
	); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) GetSpendableById(btcTxId ethcommon.Hash) (*Spendable, bool, error) {
	stmt, err := db.stmtcache.Prepare(queryGetSpendableById)
	if err != nil {
		return nil, false, err
	}

	var sqlSpendable sqlSpendable
	if err := stmt.QueryRow(btcTxId.String()[2:]).Scan(
		&sqlSpendable.BtcTxId,
		&sqlSpendable.Idx,
		&sqlSpendable.Amount,
		&sqlSpendable.BlockNumber,
		&sqlSpendable.Lock,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	spendable, err := sqlSpendable.decode()
	if err != nil {
		return nil, false, err
	}

	return spendable, true, nil
}

func (db *BtcWalletDB) SetLockOnSpendable(btcTxId ethcommon.Hash, lock bool) error {
	stmt, err := db.stmtcache.Prepare(querySetLockOnSpendable)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		lock,
		btcTxId.String()[2:],
	); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) DeleteSpendable(btcTxId ethcommon.Hash) error {
	stmt, err := db.stmtcache.Prepare(queryDeleteSpendable)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(btcTxId.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) RequestSpendablesByAmount(amount *big.Int) ([]*Spendable, bool, error) {
	stmt, err := db.stmtcache.Prepare(queryGetSortedSpendables)
	if err != nil {
		return nil, false, err
	}

	spendables := []*Spendable{}
	sum := big.NewInt(0)

	rows, err := stmt.Query()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	for rows.Next() {
		// return if having collected enough funds
		if sum.Cmp(amount) != -1 {
			break
		}

		var sqlSpendable sqlSpendable
		if err := rows.Scan(
			&sqlSpendable.BtcTxId,
			&sqlSpendable.Idx,
			&sqlSpendable.Amount,
			&sqlSpendable.BlockNumber,
			&sqlSpendable.Lock,
		); err != nil {
			return nil, false, err
		}

		spendable, err := sqlSpendable.decode()
		if err != nil {
			return nil, false, err
		}
		spendables = append(spendables, spendable)
		sum = sum.Add(sum, spendable.Amount)
	}

	ok := sum.Cmp(amount) != -1
	if !ok {
		return nil, false, nil
	}
	return spendables, true, nil
}

func (db *BtcWalletDB) InsertRequest(req *Request) error {
	stmt, err := db.stmtcache.Prepare(queryInsertRequest)
	if err != nil {
		return err
	}

	var sqlRequest sqlRequest
	if _, err := sqlRequest.encode(req); err != nil {
		return err
	}

	if _, err := stmt.Exec(
		sqlRequest.Id,
		sqlRequest.Outpoints,
		sqlRequest.Status,
	); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) GetRequestById(id ethcommon.Hash) (*Request, bool, error) {
	stmt, err := db.stmtcache.Prepare(queryGetRequestById)
	if err != nil {
		return nil, false, err
	}

	var sqlRequest sqlRequest
	if err := stmt.QueryRow(id.String()[2:]).Scan(
		&sqlRequest.Id,
		&sqlRequest.Outpoints,
		&sqlRequest.CreatedAt,
		&sqlRequest.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	req, err := sqlRequest.decode()
	if err != nil {
		return nil, false, err
	}

	return req, true, nil
}

func (db *BtcWalletDB) DeleteRequest(id ethcommon.Hash) error {
	stmt, err := db.stmtcache.Prepare(queryDeleteRequest)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(id.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) UpdateRequestStatus(id ethcommon.Hash, status RequestStatus) error {
	stmt, err := db.stmtcache.Prepare(queryUpdateRequestStatus)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		status,
		id.String()[2:],
	); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) GetRequestsByStatus(status RequestStatus) ([]*Request, error) {
	stmt, err := db.stmtcache.Prepare(queryGetRequestsByStatus)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(string(status))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	requests := []*Request{}
	for rows.Next() {
		var sqlRequest sqlRequest
		if err := rows.Scan(
			&sqlRequest.Id,
			&sqlRequest.Outpoints,
			&sqlRequest.CreatedAt,
			&sqlRequest.Status,
		); err != nil {
			return nil, err
		}

		req, err := sqlRequest.decode()
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, nil
}
