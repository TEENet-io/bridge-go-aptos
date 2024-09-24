package state

import (
	"database/sql"
	"errors"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func (stdb *StateDB) InsertMint(m *Mint) error {
	query := `INSERT INTO mint (btcTxId, receiver, amount, status) VALUES (?, ?, ?, ?)`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s := &sqlMint{}
	s, err = s.encode(m)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(s.BtcTxID, s.Receiver, s.Amount, string(MintStatusRequested))
	return err
}

func (stdb *StateDB) GetRequestedMint() ([]*Mint, error) {
	query := `SELECT btcTxId, receiver, amount, status FROM mint WHERE status = ?`
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
		if err := rows.Scan(&s.BtcTxID, &s.Receiver, &s.Amount, &s.Status); err != nil {
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

func (stdb *StateDB) GetMint(btcTxId ethcommon.Hash, status MintStatus) (*Mint, bool, error) {
	var query string
	if status == MintStatusRequested {
		query = `SELECT btcTxId, receiver, amount FROM mint WHERE btcTxId = ? AND status = ?`
	} else {
		query = `SELECT btcTxId, mintTxHash, receiver, amount FROM mint WHERE btcTxId = ? AND status = ?`
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
			&s.BtcTxID, &s.MintTxHash, &s.Receiver, &s.Amount)
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

func (stdb *StateDB) UpdateMint(m *Mint) error {
	_, ok, err := stdb.GetMint(m.BtcTxID, MintStatusRequested)
	if err != nil {
		return err
	}

	if !ok {
		msg := fmt.Sprintf("mint not found in statedb for btcTxId=%v", m.BtcTxID)
		return errors.New(msg)
	}

	query := `UPDATE mint SET mintTxHash = ?, status = ? WHERE btcTxId = ?`

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s := &sqlMint{}
	s, err = s.encode(m)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(s.MintTxHash, string(MintStatusCompleted), s.BtcTxID)

	if err != nil {
		return err
	}
	return nil
}
