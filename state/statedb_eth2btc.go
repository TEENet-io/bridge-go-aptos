package state

import (
	"database/sql"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func (st *StateDB) InsertAfterRequested(redeem *Redeem) error {
	// Insert after receiving a new redeem requested event. Only fields
	// requestTxHash, requester, receiver, amount, and status are required.
	query := `INSERT OR IGNORE INTO redeem (` + statusRequestedParamList + `) VALUES (?, ?, ?, ?, ?)`

	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	r := &sqlRedeem{}
	r, err = r.encode(redeem)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(
		r.RequestTxHash,
		r.Requester,
		r.Receiver,
		r.Amount,
		r.Status,
	); err != nil {
		return err
	}

	return nil
}

func (st *StateDB) UpdateAfterPrepared(redeem *Redeem) error {
	// Update after receiving a new redeem prepared event. Only fields
	// prepareTxHash, outpoints, and status are required.
	var query string
	_, ok, err := st.GetRedeem(redeem.RequestTxHash)
	if err != nil {
		return err
	}
	if ok {
		query = `UPDATE redeem SET prepareTxHash = ?, outpoints = ?, status = ? WHERE requestTxHash = ?`
	} else {
		query = `INSERT OR IGNORE INTO redeem (` + statusPreparedParamList + `) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}

	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	r := &sqlRedeem{}
	r, err = r.encode(redeem)
	if err != nil {
		return err
	}

	if ok {
		if _, err := stmt.Exec(r.PrepareTxHash, r.Outpoints, r.Status, r.RequestTxHash); err != nil {
			return err
		}
	} else {
		if _, err := stmt.Exec(
			r.RequestTxHash,
			r.PrepareTxHash,
			r.Requester,
			r.Receiver,
			r.Amount,
			r.Outpoints,
			r.Status,
		); err != nil {
			return err
		}
	}

	return nil
}

func (st *StateDB) GetRedeemsByStatus(status RedeemStatus) ([]*Redeem, error) {
	query := `SELECT * FROM redeem WHERE status = ?`

	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return []*Redeem{}, err
	}

	rows, err := stmt.Query(status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No rows found, return nil slice
		}
		return nil, err
	}
	defer rows.Close()

	var (
		prepareTxHash, btcTxId sql.NullString
		redeems                []*Redeem
	)

	for rows.Next() {
		var r sqlRedeem
		if err := rows.Scan(
			&r.RequestTxHash,
			&prepareTxHash,
			&btcTxId,
			&r.Requester,
			&r.Receiver,
			&r.Amount,
			&r.Outpoints,
			&r.Status,
		); err != nil {
			return nil, err
		}

		if prepareTxHash.Valid {
			r.PrepareTxHash = prepareTxHash.String
		}

		if btcTxId.Valid {
			r.BtcTxId = btcTxId.String
		}

		redeem, err := r.decode()
		if err != nil {
			return nil, err
		}
		redeems = append(redeems, redeem)
	}

	return redeems, nil
}

func (st *StateDB) GetRedeem(requestTxHash ethcommon.Hash) (*Redeem, bool, error) {
	query := `SELECT * FROM redeem WHERE requestTxHash = ?;`

	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

	var (
		r                      sqlRedeem
		prepareTxHash, btcTxId sql.NullString
	)

	row := stmt.QueryRow(requestTxHash.String()[2:])
	if err := row.Scan(
		&r.RequestTxHash,
		&prepareTxHash,
		&btcTxId,
		&r.Requester,
		&r.Receiver,
		&r.Amount,
		&r.Outpoints,
		&r.Status,
	); err != nil {
		if err == sql.ErrNoRows { // no redeem found
			return nil, false, nil
		}
		return nil, false, err
	}

	if prepareTxHash.Valid {
		r.PrepareTxHash = prepareTxHash.String
	}

	if btcTxId.Valid {
		r.BtcTxId = btcTxId.String
	}

	redeem, err := r.decode()
	if err != nil {
		return nil, false, err
	}

	return redeem, true, nil
}

func (st *StateDB) HasRedeem(requestTxHash ethcommon.Hash) (bool, RedeemStatus, error) {
	query := `SELECT status FROM redeem WHERE requestTxHash = ?`
	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return false, "", err
	}

	hash := requestTxHash.String()[2:]
	var status string
	if err := stmt.QueryRow(hash).Scan(&status); err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}

	return true, RedeemStatus(status), nil
}
