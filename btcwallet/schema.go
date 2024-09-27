package btcwallet

import "strings"

var (
	strZeroBytes32 = strings.Repeat("0", 64)

	spendableTable = `CREATE TABLE IF NOT EXISTS spendable (
		btcTxId CHAR(64) PRIMARY KEY NOT NULL,
		idx INT NOT NULL,
		amount BIGINT NOT NULL,
		blockNumber BIGINT NOT NULL,
		lock BOOLEAN NOT NULL DEFAULT FALSE,
		CONSTRAINT chk_btcTxId CHECK (btcTxId != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_idx CHECK (idx >= 0),
		CONSTRAINT chk_amount CHECK (amount > 0),
		CONSTRAINT chk_blockNumber CHECK (blockNumber > 0)
	);`

	sortedSpendable = `CREATE VIEW IF NOT EXISTS sorted_spendable AS 
		SELECT btcTxId, idx, amount, blockNumber, lock FROM spendable
		WHERE lock = FALSE
		ORDER BY blockNumber ASC, amount ASC;`

	queryInsertSpendable = `INSERT INTO spendable (
		btcTxId, idx, amount, blockNumber, lock) VALUES (?, ?, ?, ?, ?);`
	querySetLockOnSpendable  = `UPDATE spendable SET lock = ? WHERE btcTxId = ?;`
	queryGetSpendableById    = `SELECT * FROM spendable WHERE btcTxId = ?;`
	queryDeleteSpendable     = `DELETE FROM spendable WHERE btcTxId = ?;`
	queryGetSortedSpendables = `SELECT * FROM sorted_spendable;`

	requestTable = `CREATE TABLE IF NOT EXISTS request (
		id CHAR(64) PRIMARY KEY NOT NULL,
		outpoints BLOB NOT NULL,
		createdAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_id CHECK (id != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_status CHECK (status IN ('locked', 'timeout', 'spent'))
	);`
	queryInsertRequest       = `INSERT INTO request (id, outpoints, status) VALUES (?, ?, ?);`
	queryGetRequestById      = `SELECT * FROM request WHERE id = ?;`
	queryDeleteRequest       = `DELETE FROM request WHERE id = ?;`
	queryUpdateRequestStatus = `UPDATE request SET status = ? WHERE id = ?;`
	queryGetRequestsByStatus = `SELECT * FROM request WHERE status = ?;`
)
