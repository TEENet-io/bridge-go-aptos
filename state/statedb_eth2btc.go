package state

import (
	"database/sql"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// Insert after receiving a new redeem requested event (from user). Only fields
// requestTxHash, requester, receiver, amount, and status are required.
func (stdb *StateDB) InsertAfterRequested(redeem *Redeem) error {
	query := `INSERT OR IGNORE INTO redeem (` + statusRequestedParamList + `) VALUES (?, ?, ?, ?, ?)`

	stmt, err := stdb.stmtCache.Prepare(query)
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

// Update the db, after receiving a new "RedeemPrepared" event.
// New fields to update to said (found) redeem are:
// prepareTxHash, outpoints, and status.
func (stdb *StateDB) UpdateAfterPrepared(redeem *Redeem) error {
	var query string
	// "ok" marks if the redeems is found in the database.
	_, ok, err := stdb.GetRedeem(redeem.RequestTxHash)
	if err != nil {
		return err
	}
	if ok {
		query = `UPDATE redeem SET prepareTxHash = ?, outpoints = ?, status = ? WHERE requestTxHash = ?`
	} else {
		query = `INSERT OR IGNORE INTO redeem (` + statusPreparedParamList + `) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}

	stmt, err := stdb.stmtCache.Prepare(query)
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

// UpdateAfterRedeemed updates the database row,
// after a redeem is sent out on BTC side.
// It uses requestTxHash (from redeem param) to look for a redeem record in the database.
// Then writes in the btcTxId (from redeem param) and set status to completed on the database record.
func (stdb *StateDB) UpdateAfterRedeemed(redeem *Redeem) error {
	query := `UPDATE redeem SET btcTxId = ?, status = ? WHERE requestTxHash = ?`

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	r := &sqlRedeem{}
	r, err = r.encode(redeem)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(r.BtcTxId, r.Status, r.RequestTxHash); err != nil {
		return err
	}

	return nil
}

// Query Redeem from the database by "status".
func (stdb *StateDB) GetRedeemsByStatus(status RedeemStatus) ([]*Redeem, error) {
	query := `SELECT * FROM redeem WHERE status = ?`

	stmt, err := stdb.stmtCache.Prepare(query)
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
			&prepareTxHash, // temp fill, fill later.
			&btcTxId,       // temp fill, fill later
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

// Query Redeem from database via requestTxHash.
// Return (*Redeem, bool: found/not found, error)
func (stdb *StateDB) GetRedeem(requestTxHash ethcommon.Hash) (*Redeem, bool, error) {
	query := `SELECT * FROM redeem WHERE requestTxHash = ?;`

	stmt, err := stdb.stmtCache.Prepare(query)
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

func (stdb *StateDB) GetRedeemsByRequester(requester string) ([]*Redeem, error) {
	query := `SELECT * FROM redeem WHERE LOWER(requester) = LOWER(?)`

	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return nil, err
	}

	// Strip "0x" prefix off the string.
	requesterStr := common.Trim0xPrefix(requester)

	rows, err := stmt.Query(requesterStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
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
			&prepareTxHash, // temp fill, fill later.
			&btcTxId,       // temp fill, fill later
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

// Query if a redeem exists in the database via requestTxHash.
// Return (bool: found/not found, RedeemStatus, error)
func (stdb *StateDB) HasRedeem(requestTxHash ethcommon.Hash) (bool, RedeemStatus, error) {
	query := `SELECT status FROM redeem WHERE requestTxHash = ?`
	stmt, err := stdb.stmtCache.Prepare(query)
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
