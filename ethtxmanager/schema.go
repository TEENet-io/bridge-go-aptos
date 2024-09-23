package ethtxmanager

import "strings"

var (
	strZeroBytes32 = strings.Repeat("0", 64)

	signatureRequestTable = `CREATE TABLE IF NOT EXISTS signatureRequest (
		requestTxHash CHAR(64) PRIMARY KEY NOT NULL,
		signingHash CHAR(64) UNIQUE NOT NULL,
		rx CHAR(64),
		s CHAR(64),
		CONSTRAINT chk_requestTxHash CHECK (requestTxHash != '` + strZeroBytes32 + `')
		CONSTRAINT chk_signingHash CHECK (signingHash != '` + strZeroBytes32 + `')
	);`

	// sentAfter == hash of the latest block before sending the tx
	monitoredTxTable = `CREATE TABLE IF NOT EXISTS monitoredTx (
		txHash CHAR(64) PRIMARY KEY NOT NULL,
		requestTxHash CHAR(64) UNIQUE NOT NULL,
		sentAfter CHAR(64) NOT NULL,
		CONSTRAINT chk_txHash CHECK (txHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_requestTxHash CHECK (requestTxHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_sentAfter CHECK (sentAfter != '` + strZeroBytes32 + `')
	);`
)
