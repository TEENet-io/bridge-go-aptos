package state

import (
	"database/sql"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/database"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type StateDB struct {
	stmtCache *database.StmtCache
}

func NewStateDB(db *sql.DB) (*StateDB, error) {
	// 1. Create the tables.
	if _, err := db.Exec(redeemTable + kvTable + mintTable); err != nil {
		return nil, err
	}

	// 2. A stmt cache + db.
	return &StateDB{
		stmtCache: database.NewStmtCache(db),
	}, nil
}

func (stdb *StateDB) Close() {
	stdb.stmtCache.Clear()
}

func (stdb *StateDB) GetKeyedValue(key ethcommon.Hash) (ethcommon.Hash, bool, error) {
	query := `SELECT value FROM kv WHERE key = ?`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return ethcommon.Hash{}, false, err
	}

	var value string
	keyHex := key.String()[2:]
	if err := stmt.QueryRow(keyHex).Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return ethcommon.Hash{}, false, nil
		}
		return ethcommon.Hash{}, false, err
	}

	return common.HexStrToBytes32(value), true, nil
}

func (stdb *StateDB) SetKeyedValue(key, value ethcommon.Hash) error {
	query := `INSERT OR REPLACE INTO kv (key, value) VALUES (?, ?)`
	stmt, err := stdb.stmtCache.Prepare(query)
	if err != nil {
		return err
	}

	keyHex := key.String()[2:]
	valueHex := value.String()[2:]
	if _, err := stmt.Exec(keyHex, valueHex); err != nil {
		return err
	}

	return nil
}
