package ethtxmanager

import (
	"database/sql"

	"github.com/TEENet-io/bridge-go/database"
)

type EthTxManagerDB struct {
	db        *sql.DB
	stmtCache *database.StmtCache
}

func newEthTxManagerDB(db *sql.DB) (*EthTxManagerDB, error) {
	if _, err := db.Exec(toPrepareTable); err != nil {
		return nil, err
	}

	return &EthTxManagerDB{
		db:        db,
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (mgrdb *EthTxManagerDB) close() {
	mgrdb.stmtCache.Clear()
}
