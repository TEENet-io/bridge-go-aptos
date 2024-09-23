package eth2btcstate

import (
	"database/sql"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type StateDB struct {
	db        *sql.DB
	stmtCache *database.StmtCache
}

func NewStateDB(db *sql.DB) (*StateDB, error) {
	if _, err := db.Exec(redeemTable + kvTable); err != nil {
		return nil, err
	}

	return &StateDB{
		db:        db,
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (st *StateDB) Close() {
	st.stmtCache.Clear()
}

func (st *StateDB) insertAfterRequested(redeem *Redeem) error {
	// Insert after receiving a new redeem requested event. Only fields
	// requestTxHash, requester, receiver, amount, and status are required.
	query := `INSERT OR IGNORE INTO redeem (` + statusRequestedParamList + `) VALUES (?, ?, ?, ?, ?)`

	stmt, err := st.stmtCache.Prepare(query)

	r, err := encode(redeem)
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

func (st *StateDB) updateAfterPrepared(redeem *Redeem) error {
	// Update after receiving a new redeem prepared event. Only fields
	// prepareTxHash, outpoints, and status are required.
	var query string
	_, ok, err := st.Get(redeem.RequestTxHash, RedeemStatusRequested)
	if err != nil {
		return err
	}
	if ok {
		query = `UPDATE redeem SET prepareTxHash = ?, outpoints = ?, status = ? WHERE requestTxHash = ?`
	} else {
		query = `INSERT OR IGNORE INTO redeem (` + statusPreparedParamList + `) VALUES (?, ?, ?, ?, ?, ?, ?)`
	}

	stmt, err := st.stmtCache.Prepare(query)

	r, err := encode(redeem)
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

func (st *StateDB) GetByStatus(status RedeemStatus) ([]*Redeem, error) {
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

func (st *StateDB) Get(requestTxHash ethcommon.Hash, status RedeemStatus) (*Redeem, bool, error) {
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

func (st *StateDB) Has(requestTxHash ethcommon.Hash) (bool, RedeemStatus, error) {
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

func (st *StateDB) GetKeyedValue(key ethcommon.Hash) (ethcommon.Hash, error) {
	query := `SELECT value FROM kv WHERE key = ?`
	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return ethcommon.Hash{}, err
	}

	var value string
	keyHex := key.String()[2:]
	if err := stmt.QueryRow(keyHex).Scan(&value); err != nil {
		return [32]byte{}, err
	}

	return common.HexStrToBytes32(value), nil
}

func (st *StateDB) setKeyedValue(key, value ethcommon.Hash) error {
	query := `INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)`
	stmt, err := st.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	keyHex := key.String()[2:]
	valueHex := value.String()[2:]
	if _, err := stmt.Exec(keyHex, valueHex); err != nil {
		return err
	}

	return nil
}
