package btc2ethstate

import (
	"database/sql"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestOps(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	stdb, err := NewStateDB(db)
	assert.NoError(t, err)

	// no record
	rows, err := stdb.GetRequested()
	assert.NoError(t, err)
	assert.Empty(t, rows)

	// insert a requested mint
	expected := RandMint(100, 2, MintStatusRequested)
	err = stdb.Insert(expected)
	assert.NoError(t, err)

	// no record
	_, ok, err := stdb.Get(expected.BtcTxID, MintStatusCompleted)
	assert.NoError(t, err)
	assert.False(t, ok)

	// record exists with status = requested
	actual, ok, err := stdb.Get(expected.BtcTxID, MintStatusRequested)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected.BtcTxID, actual.BtcTxID)
	assert.Equal(t, expected.Receiver, actual.Receiver)
	assert.Equal(t, expected.Amount, actual.Amount)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Empty(t, actual.Outpoints)
	assert.Equal(t, ethcommon.Hash{}, actual.MintTxHash)

	// insert and update another mint
	expected = RandMint(200, 3, MintStatusCompleted)
	err = stdb.Insert(expected)
	assert.NoError(t, err)
	err = stdb.Update(expected)
	assert.NoError(t, err)
	actual, ok, err = stdb.Get(expected.BtcTxID, MintStatusCompleted)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected.String(), actual.String())

	// check mints with status = requested
	rows, err = stdb.GetRequested()
	assert.NoError(t, err)
	assert.Len(t, rows, 1)
}
