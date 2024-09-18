package eth2btcstate

import (
	"database/sql"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

type StateDB struct {
	db        *sql.DB
	stmtCache *StmtCache
}

var stateDBErrors StateDBError

func NewStateDB(driverName, dataSourceName string) (*StateDB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()

	if _, err := db.Exec(redeemTable + kvTable); err != nil {
		return nil, err
	}

	return &StateDB{
		db:        db,
		stmtCache: NewStmtCache(db),
	}, nil
}

func (st *StateDB) Close() error {
	st.stmtCache.Clear()

	if err := st.db.Close(); err != nil {
		return err
	}
	return nil
}

func (st *StateDB) InsertAfterRequested(redeem *Redeem) error {
	if redeem.Status != RedeemStatusRequested && redeem.Status != RedeemStatusInvalid {
		return stateDBErrors.CannotInsertDueToInvalidStatus(redeem)
	}

	// Insert after receiving a new redeem requested event. Only fields
	// requestTxHash, requester, receiver, amount, and status are required.
	query := `INSERT OR IGNORE INTO redeem (
		requestTxHash,
		requester,
		receiver,
		amount,
		status
	) VALUES (?, ?, ?, ?, ?)`

	stmt := st.stmtCache.MustPrepare(query)

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

func (st *StateDB) UpdateAfterPrepared(redeem *Redeem) error {
	if redeem.Status != RedeemStatusPrepared {
		return stateDBErrors.CannotUpdateDueToInvalidStatus(redeem)
	}

	// Update after receiving a new redeem prepared event. Only fields
	// prepareTxHash, outpoints, and status are required.
	query := `UPDATE redeem SET prepareTxHash = ?, outpoints = ?, status = ? WHERE requestTxHash = ?`

	stmt := st.stmtCache.MustPrepare(query)

	r, err := encode(redeem)
	if err != nil {
		return err
	}

	if _, err := stmt.Exec(r.PrepareTxHash, r.Outpoints, r.Status, r.RequestTxHash); err != nil {
		return err
	}

	return nil
}

func (st *StateDB) GetByStatus(status RedeemStatus) ([]*Redeem, error) {
	var query string
	if status == RedeemStatusRequested || status == RedeemStatusInvalid {
		query = `SELECT requestTxHash, requester, receiver, amount, status FROM redeem WHERE status = ?`
	} else if status == RedeemStatusPrepared {
		query = `SELECT 
			requestTxHash, prepareTxHash, requester, receiver, amount, outpoints, status 
		FROM redeem WHERE status = ?`
	} else {
		query = `SELECT * FROM redeem WHERE status = ?`
	}
	stmt := st.stmtCache.MustPrepare(query)

	rows, err := stmt.Query(status)
	if err != nil {
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

func (st *StateDB) Get(requestTxHash []byte, status RedeemStatus) (redeem *Redeem, err error) {
	var query string
	if status == RedeemStatusRequested || status == RedeemStatusInvalid {
		query = `SELECT 
			requestTxHash, 
			requester, 
			receiver, 
			amount, 
			status 
		FROM redeem WHERE requestTxHash = ? AND status = ?`
	} else if status == RedeemStatusPrepared {
		query = `SELECT 
			requestTxHash, 
			prepareTxHash, 
			requester, 
			receiver, 
			amount, 
			outpoints, 
			status 
		FROM redeem WHERE requestTxHash = ? AND status = ?`
	} else {
		query = `SELECT * FROM redeem WHERE requestTxHash = ? AND status = ?`
	}
	stmt := st.stmtCache.MustPrepare(query)

	hash := ethcommon.Bytes2Hex(requestTxHash)
	row := stmt.QueryRow(hash, string(status))
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
			return nil, nil
		}
		return nil, err
	}

	redeem, err = r.decode()
	if err != nil {
		return nil, err
	}

	return
}

func (st *StateDB) Has(requestTxHash []byte) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM redeem WHERE requestTxHash = ?)`
	stmt := st.stmtCache.MustPrepare(query)

	hash := ethcommon.Bytes2Hex(requestTxHash)
	var exists bool
	if err := stmt.QueryRow(hash).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (st *StateDB) KVGet(key []byte) ([]byte, error) {
	query := `SELECT value FROM kv WHERE key = ?`
	stmt := st.stmtCache.MustPrepare(query)

	var value string
	keyHex := ethcommon.Bytes2Hex(ethcommon.LeftPadBytes(key, 32))
	if err := stmt.QueryRow(keyHex).Scan(&value); err != nil {
		return nil, err
	}

	return ethcommon.Hex2BytesFixed(value, 32), nil
}

func (st *StateDB) KVSet(key, value []byte) error {
	query := `INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)`
	stmt := st.stmtCache.MustPrepare(query)

	keyHex := ethcommon.Bytes2Hex(ethcommon.LeftPadBytes(key, 32))
	valueHex := ethcommon.Bytes2Hex(ethcommon.LeftPadBytes(value, 32))
	if _, err := stmt.Exec(keyHex, valueHex); err != nil {
		return err
	}

	return nil
}
