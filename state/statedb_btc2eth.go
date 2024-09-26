package state

import (
	"database/sql"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func (stdb *StateDB) InsertMint(m *Mint) error {
	query := `INSERT INTO mint (BtcTxId, mintTxHash, receiver, amount) VALUES (?, ?, ?, ?)`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s := &sqlMint{}
	s, err = s.encode(m)
	if err != nil {
		return err
	}

	var mintTxHash sql.NullString
	if m.MintTxHash != common.EmptyHash {
		mintTxHash.String = m.MintTxHash.String()[2:]
		mintTxHash.Valid = true
	} else {
		mintTxHash.Valid = false
	}

	_, err = stmt.Exec(s.BtcTxId, mintTxHash, s.Receiver, s.Amount)
	return err
}

func (stdb *StateDB) GetUnMinted() ([]*Mint, error) {
	query := `SELECT BtcTxId, receiver, amount FROM mint WHERE mintTxHash IS NULL`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	mints := []*Mint{}
	for rows.Next() {
		var s sqlMint
		if err := rows.Scan(&s.BtcTxId, &s.Receiver, &s.Amount); err != nil {
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

func (stdb *StateDB) GetMint(BtcTxId ethcommon.Hash) (*Mint, bool, error) {
	query := `SELECT BtcTxId, mintTxHash, receiver, amount FROM mint WHERE BtcTxId = ?`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

	var (
		s          sqlMint
		mintTxHash sql.NullString
	)

	id := BtcTxId.String()[2:]
	if err := stmt.QueryRow(id).Scan(&s.BtcTxId, &mintTxHash, &s.Receiver, &s.Amount); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	if mintTxHash.Valid {
		s.MintTxHash = mintTxHash.String
	}

	mint, err := s.decode()
	if err != nil {
		return nil, false, err
	}

	return mint, true, nil
}

// UpdateMint updates the mint with the given BtcTxId
// If the mint does not exist, it inserts a new row
func (stdb *StateDB) UpdateMint(m *Mint) error {
	_, ok, err := stdb.GetMint(m.BtcTxId)
	if err != nil {
		return err
	}
	if !ok {
		return stdb.InsertMint(m)
	}

	query := `UPDATE mint SET mintTxHash = ? WHERE BtcTxId = ?`

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	s := &sqlMint{}
	s, err = s.encode(m)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(s.MintTxHash, s.BtcTxId)

	if err != nil {
		return err
	}
	return nil
}
