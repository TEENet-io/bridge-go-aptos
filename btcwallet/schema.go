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

	queryInsert = `INSERT INTO spendable (
		btcTxId, idx, amount, blockNumber, lock) VALUES (?, ?, ?, ?, ?);`
	querySetLock   = `UPDATE spendable SET lock = ? WHERE btcTxId = ?;`
	queryGetById   = `SELECT * FROM spendable WHERE btcTxId = ?;`
	queryDelete    = `DELETE FROM spendable WHERE btcTxId = ?;`
	queryGetSorted = `SELECT * FROM sorted_spendable;`
)
