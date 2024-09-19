package ethtxmanager

import "strings"

var (
	zeroBytes32 = "0x" + strings.Repeat("0", 64)

	toPrepareTable = `CREATE TABLE IF NOT EXISTS toPrepare (
		requestTxHash CHAR(64) PRIMARY KEY NOT NULL,
		createdAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT chk_requestTxHash CHECK (requestTxHash != '` + zeroBytes32 + `')
	);`
)
