package state

import (
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestKV(t *testing.T) {
	sqlDB := getMemoryDB()
	db, err := NewStateDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB.Close()
		db.Close()
	}()

	// insert
	key := ethcommon.Hash{}
	key.SetBytes([]byte("key"))
	val := ethcommon.Hash{}
	val.SetBytes([]byte("value1"))
	err = db.SetKeyedValue(key, val)
	assert.NoError(t, err)

	// get
	v, err := db.GetKeyedValue(key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), ethcommon.TrimLeftZeroes(v[:]))

	val.SetBytes([]byte("value2"))
	err = db.SetKeyedValue(key, val)
	assert.NoError(t, err)
	v, err = db.GetKeyedValue(key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), ethcommon.TrimLeftZeroes(v[:]))
}
