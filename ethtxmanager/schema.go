package ethtxmanager

import "strings"

var (
	strZeroBytes32 = strings.Repeat("0", 64)

	// sentAfter == hash of the latest block before sending the tx
	MonitoredTxTable = `CREATE TABLE IF NOT EXISTS MonitoredTx (
		txHash CHAR(64) PRIMARY KEY NOT NULL,
		id CHAR(64) NOT NULL,
		sentAfter CHAR(64) NOT NULL,
		minedAt CHAR(64),
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_txHash CHECK (txHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_id CHECK (id != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_sentAfter CHECK (sentAfter != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_minedAt CHECK (minedAt IS NULL OR minedAt != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_status CHECK (status IN ('pending', 'timeout', 'success', 'reverted', 'reorg'))
	);`

	queryInsertPendingMonitoredTx = `INSERT INTO MonitoredTx (
		txHash, id, sentAfter, status) VALUES (?,?,?,?);`
	queryInsertMonitoredTx = `INSERT INTO MonitoredTx (
		txHash, id, sentAfter, minedAt, status) VALUES (?,?,?,?,?);`
	queryUpdateMonitoredTxStatus     = `UPDATE MonitoredTx SET status = ? WHERE txHash = ?;`
	queryUpdateMonitoredTxAfterMined = `UPDATE MonitoredTx SET minedAt = ?, status = ? WHERE txHash = ?;`
	queryGetMonitoredTxByTxHash      = `SELECT * FROM MonitoredTx WHERE txHash = ?;`
	queryGetMonitoredTxsById         = `SELECT * FROM MonitoredTx WHERE id = ?;`
	queryGetMonitoredTxsByStatus     = `SELECT * FROM MonitoredTx WHERE status = ?;`
	queryDeleteMonitoredTxByTxHash   = `DELETE FROM MonitoredTx WHERE txHash = ?;`
)
