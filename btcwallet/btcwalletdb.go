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
	if _, err := db.Exec(spendableTable + sortedSpendable); err != nil {
		return nil, err
	}

	return &BtcWalletDB{
		stmtcache: database.NewStmtCache(db),
	}, nil
}

func (db *BtcWalletDB) Close() {
	db.stmtcache.Clear()
}

func (db *BtcWalletDB) Insert(spendable *Spendable) error {
	stmt, err := db.stmtcache.Prepare(queryInsert)
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

func (db *BtcWalletDB) GetById(btcTxId ethcommon.Hash) (*Spendable, bool, error) {
	stmt, err := db.stmtcache.Prepare(queryGetById)
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

func (db *BtcWalletDB) SetLock(btcTxId ethcommon.Hash, lock bool) error {
	stmt, err := db.stmtcache.Prepare(querySetLock)
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

func (db *BtcWalletDB) Delete(btcTxId ethcommon.Hash) error {
	stmt, err := db.stmtcache.Prepare(queryDelete)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(btcTxId.String()[2:]); err != nil {
		return err
	}

	return nil
}

func (db *BtcWalletDB) GetSpendables(amount *big.Int) ([]*Spendable, bool, error) {
	stmt, err := db.stmtcache.Prepare(queryGetSorted)
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
