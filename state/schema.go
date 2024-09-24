package state

import "strings"

var (
	strZeroBytes32 = strings.Repeat("0", 64)
	strZeroBytes20 = strings.Repeat("0", 40)

	// table that stores the life cycle of a redeem request
	redeemTable = `CREATE TABLE IF NOT EXISTS redeem (
		requestTxHash CHAR(64) PRIMARY KEY NOT NULL,
		prepareTxHash CHAR(64) UNIQUE,
		btcTxId CHAR(64) UNIQUE,
		requester CHAR(40) NOT NULL,
		receiver VARCHAR(62) NOT NULL,
		amount BIGINT UNSIGNED NOT NULL,
		outpoints BLOB,
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_status CHECK (status IN ('requested', 'prepared', 'redeemed', 'invalid')),
		CONSTRAINT chk_amount CHECK (amount > 0)
		CONSTRAINT chk_requestTxHash CHECK (requestTxHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_prepareTxHash CHECK (prepareTxHash IS NULL OR prepareTxHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_btcTxId CHECK (btcTxId IS NULL OR btcTxId != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_requester CHECK (requester != '` + strZeroBytes20 + `')
	);`

	// table stores key-value pairs. Both key and value are a 32-byte hex string without prefix '0x'
	kvTable = `CREATE TABLE IF NOT EXISTS kv (
		key CHAR(64) PRIMARY KEY NOT NULL,
		value CHAR(64) NOT NULL
	);`

	mintTable = `CREATE TABLE IF NOT EXISTS mint (
		btcTxId CHAR(64) PRIMARY KEY NOT NULL,
		mintTxHash CHAR(64) UNIQUE,
		receiver CHAR(40) NOT NULL,
		amount BIGINT UNSIGNED NOT NULL,
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_status CHECK (status IN ('requested', 'completed')),
		CONSTRAINT chk_amount CHECK (amount > 0),
		CONSTRAINT chk_btcTxId CHECK (btcTxId != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_receiver CHECK (receiver != '` + strZeroBytes20 + `')
	);`

	statusRequestedParamList = " requestTxHash, requester, receiver, amount, status "
	statusPreparedParamList  = " requestTxHash, prepareTxHash, requester, receiver, amount, outpoints, status "
)
