package state

import (
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestStateDBOps(t *testing.T) {
	sqlDB := getMemoryDB()
	defer sqlDB.Close()
	stdb, err := NewStateDB(sqlDB)
	assert.NoError(t, err)
	defer stdb.Close()

	mints, err := stdb.GetRequestedMint()
	assert.NoError(t, err)
	assert.Len(t, mints, 0)

	expected := randMint(MintStatusRequested)

	err = stdb.InsertMint(expected)
	assert.NoError(t, err)

	mints, err = stdb.GetRequestedMint()
	assert.NoError(t, err)
	assert.Len(t, mints, 1)
	assert.Equal(t, expected.BtcTxID, mints[0].BtcTxID)
	assert.Equal(t, expected.Receiver, mints[0].Receiver)
	assert.Equal(t, expected.Amount, mints[0].Amount)
	assert.Equal(t, MintStatusRequested, mints[0].Status)
	assert.Equal(t, ethcommon.Hash{}, mints[0].MintTxHash)

	_, ok, err := stdb.GetMint(expected.BtcTxID, MintStatusCompleted)
	assert.NoError(t, err)
	assert.False(t, ok)
	m, ok, err := stdb.GetMint(expected.BtcTxID, MintStatusRequested)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mints[0].String(), m.String())

	mint1 := randMint(MintStatusRequested)
	err = stdb.UpdateMint(mint1)
	assert.Error(t, err)

	expected.Status = MintStatusCompleted
	err = stdb.UpdateMint(expected)
	assert.NoError(t, err)
	m, ok, err = stdb.GetMint(expected.BtcTxID, MintStatusCompleted)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected.String(), m.String())
}
