package eth2btcstate

import "strings"

var (
	zeroBytes32 = "0x" + strings.Repeat("0", 64)
	zeroBytes20 = strings.Repeat("0", 40)

	// table that stores the life cycle of a redeem request
	redeemTable = `CREATE TABLE IF NOT EXISTS redeem (
		requestTxHash CHAR(66) PRIMARY KEY NOT NULL,
		prepareTxHash CHAR(66) UNIQUE,
		btcTxId CHAR(66) UNIQUE,
		requester VARCHAR(42) NOT NULL,
		receiver VARCHAR(62) NOT NULL,
		amount BIGINT UNSIGNED NOT NULL,
		outpoints BLOB,
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_status CHECK (status IN ('requested', 'prepared', 'redeemed', 'invalid')),
		CONSTRAINT chk_amount CHECK (amount > 0)
		CONSTRAINT chk_requestTxHash CHECK (requestTxHash != '` + zeroBytes32 + `'),
		CONSTRAINT chk_prepareTxHash CHECK (prepareTxHash IS NULL OR prepareTxHash != '` + zeroBytes32 + `'),
		CONSTRAINT chk_btcTxId CHECK (btcTxId IS NULL OR btcTxId != '` + zeroBytes32 + `'),
		CONSTRAINT chk_requester CHECK (requester != '` + zeroBytes20 + `')
	);`

	// table stores key-value pairs. Both key and value are a 32-byte hex string without prefix '0x'
	kvTable = `CREATE TABLE IF NOT EXISTS kv (
		key CHAR(64) PRIMARY KEY NOT NULL,
		value CHAR(64) NOT NULL
	);`

	statusRequestedParamList = " requestTxHash, requester, receiver, amount, status "
	statusPreparedParamList  = " requestTxHash, prepareTxHash, requester, receiver, amount, outpoints, status "
)
