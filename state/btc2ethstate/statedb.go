package btc2ethstate

import (
	"database/sql"
	"fmt"

	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type StateDB struct {
	stmtCache *database.StmtCache
}

func NewStateDB(db *sql.DB) (*StateDB, error) {
	if _, err := db.Exec(mintTable); err != nil {
		return nil, err
	}

	return &StateDB{
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (stdb *StateDB) Close() {
	stdb.stmtCache.Clear()
}

func (stdb *StateDB) Insert(m *Mint) error {
	query := `INSERT INTO mint (btcTxId, receiver, amount, status) VALUES (?, ?, ?, ?)`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s, err := encode(m)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(s.BtcTxID, s.Receiver, s.Amount, string(MintStatusRequested))
	return err
}

func (stdb *StateDB) GetRequested() ([]*Mint, error) {
	query := `SELECT btcTxId, receiver, amount FROM mint WHERE status = ?`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(string(MintStatusRequested))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mints := []*Mint{}
	for rows.Next() {
		var s sqlMint
		if err := rows.Scan(&s.BtcTxID, &s.Receiver, &s.Amount); err != nil {
			return nil, err
		}

		mint, err := s.decode()
		if err != nil {
			return nil, err
		}

		mints = append(mints, mint)
	}

	return mints, nil
}

func (stdb *StateDB) Get(btcTxId ethcommon.Hash, status MintStatus) (*Mint, bool, error) {
	var query string
	if status == MintStatusRequested {
		query = `SELECT btcTxId, receiver, amount FROM mint WHERE btcTxId = ? AND status = ?`
	} else {
		query = `SELECT btcTxId, mintTxHash, receiver, amount, outpoints FROM mint WHERE btcTxId = ? AND status = ?`
	}

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

	var s sqlMint
	id := btcTxId.String()[2:]
	if status == MintStatusRequested {
		s.Status = string(MintStatusRequested)
		err = stmt.QueryRow(id, string(MintStatusRequested)).Scan(&s.BtcTxID, &s.Receiver, &s.Amount)
	} else {
		s.Status = string(MintStatusCompleted)
		err = stmt.QueryRow(id, string(MintStatusCompleted)).Scan(
			&s.BtcTxID, &s.MintTxHash, &s.Receiver, &s.Amount, &s.Outpoints)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	mint, err := s.decode()
	if err != nil {
		return nil, false, err
	}

	return mint, true, nil
}

func (stdb *StateDB) Update(m *Mint) error {
	_, ok, err := stdb.Get(m.BtcTxID, MintStatusRequested)
	if err != nil {
		return err
	}

	if !ok {
		msg := fmt.Sprintf("mint not found in statedb for btcTxId=%v", m.BtcTxID)
		panic(msg)
	}

	query := `UPDATE mint SET mintTxHash = ?, outpoints = ?, status = ? WHERE btcTxId = ?`

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s, err := encode(m)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(s.MintTxHash, s.Outpoints, string(MintStatusCompleted), s.BtcTxID)

	if err != nil {
		return err
	}
	return nil
}
