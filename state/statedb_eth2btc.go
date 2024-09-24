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
	_, ok, err := st.GetRedeem(redeem.RequestTxHash, RedeemStatusRequested)
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

func (st *StateDB) GetRedeemByStatus(status RedeemStatus) ([]*Redeem, error) {
	var query string
	if status == RedeemStatusRequested || status == RedeemStatusInvalid {
		query = `SELECT` + statusRequestedParamList + `FROM redeem WHERE status = ?`
	} else if status == RedeemStatusPrepared {
		query = `SELECT` + statusPreparedParamList + `FROM redeem WHERE status = ?`
	} else {
		query = `SELECT * FROM redeem WHERE status = ?`
	}
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

	var redeems []*Redeem
	for rows.Next() {
		var r sqlRedeem
		if status == RedeemStatusRequested || status == RedeemStatusInvalid {
			if err := rows.Scan(
				&r.RequestTxHash,
				&r.Requester,
				&r.Receiver,
				&r.Amount,
				&r.Status,
			); err != nil {
				return nil, err
			}
		} else if status == RedeemStatusPrepared {
			if err := rows.Scan(
				&r.RequestTxHash,
				&r.PrepareTxHash,
				&r.Requester,
				&r.Receiver,
				&r.Amount,
				&r.Outpoints,
				&r.Status,
			); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(
				&r.RequestTxHash,
				&r.PrepareTxHash,
				&r.BtcTxId,
				&r.Requester,
				&r.Receiver,
				&r.Amount,
				&r.Outpoints,
				&r.Status,
			); err != nil {
				return nil, err
			}
		}

		redeem, err := r.decode()
		if err != nil {
			return nil, err
		}
		redeems = append(redeems, redeem)
	}

	return redeems, nil
}

func (st *StateDB) GetRedeem(requestTxHash ethcommon.Hash, status RedeemStatus) (*Redeem, bool, error) {
	var query string
	if status == RedeemStatusRequested || status == RedeemStatusInvalid {
		query = `SELECT` + statusRequestedParamList + `FROM redeem WHERE requestTxHash = ? AND status = ?`
	} else if status == RedeemStatusPrepared {
		query = `SELECT` + statusPreparedParamList + `FROM redeem WHERE requestTxHash = ? AND status = ?`
	} else {
		query = `SELECT * FROM redeem WHERE requestTxHash = ? AND status = ?`
	}
	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return nil, false, err
	}

	row := stmt.QueryRow(requestTxHash.String()[2:], string(status))
	var r sqlRedeem
	if status == RedeemStatusRequested || status == RedeemStatusInvalid {
		err = row.Scan(
			&r.RequestTxHash,
			&r.Requester,
			&r.Receiver,
			&r.Amount,
			&r.Status,
		)
	} else if status == RedeemStatusPrepared {
		err = row.Scan(
			&r.RequestTxHash,
			&r.PrepareTxHash,
			&r.Requester,
			&r.Receiver,
			&r.Amount,
			&r.Outpoints,
			&r.Status,
		)
	} else {
		err = row.Scan(
			&r.RequestTxHash,
			&r.PrepareTxHash,
			&r.BtcTxId,
			&r.Requester,
			&r.Receiver,
			&r.Amount,
			&r.Outpoints,
			&r.Status,
		)
	}

	if err != nil {
		if err == sql.ErrNoRows { // no redeem found
			return nil, false, nil
		}
		return nil, false, err
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
