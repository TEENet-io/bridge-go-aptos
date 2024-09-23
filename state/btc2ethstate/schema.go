package btc2ethstate

import "strings"

var (
	strZeroBytes32 = strings.Repeat("0", 64)
	strZeroBytes20 = strings.Repeat("0", 40)

	mintTable = `CREATE TABLE IF NOT EXISTS mint (
		btcTxID CHAR(64) PRIMARY KEY,
		mintTxHash CHAR(64) UNIQUE,
		receiver CHAR(40) NOT NULL,
		amount BIGINT NOT NULL,
		outpoints BLOB,
		status VARCHAR(10) NOT NULL,
		CONSTRAINT chk_status CHECK (status IN ('requested', 'completed')),
		CONSTRAINT chk_btcTxID CHECK (btcTxID != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_mintTxHash CHECK (mintTxHash IS NULL OR mintTxHash != '` + strZeroBytes32 + `'),
		CONSTRAINT chk_receiver CHECK (receiver != '` + strZeroBytes20 + `'),
		CONSTRAINT chk_amount CHECK (amount > 0)
	);`
)
