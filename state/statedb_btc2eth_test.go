package state

import (
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"
)

func newTestStateDB(t *testing.T) (*StateDB, func()) {
	sqlDB := getMemoryDB()
	stdb, err := NewStateDB(sqlDB)
	assert.NoError(t, err)

	close := func() {
		stdb.Close()
		sqlDB.Close()
	}

	return stdb, close
}
func TestInsertMint(t *testing.T) {
	statedb, close := newTestStateDB(t)
	defer close()

	mint := RandMint(false)
	err := statedb.InsertMint(mint)
	assert.NoError(t, err)
	chk, ok, err := statedb.GetMint(mint.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mint, chk)
}

func TestGetUnMinted(t *testing.T) {
	statedb, close := newTestStateDB(t)
	defer close()

	unminted := RandMint(false)
	err := statedb.InsertMint(unminted)
	assert.NoError(t, err)

	minted := RandMint(true)
	err = statedb.InsertMint(minted)
	assert.NoError(t, err)

	chk, err := statedb.GetUnMinted()
	assert.NoError(t, err)
	assert.Len(t, chk, 1)
	assert.Equal(t, unminted, chk[0])
}

func TestUpdateMint(t *testing.T) {
	statedb, close := newTestStateDB(t)
	defer close()

	unminted := RandMint(false)
	err := statedb.InsertMint(unminted)
	assert.NoError(t, err)

	minted := &Mint{
		BtcTxId:    unminted.BtcTxId,
		MintTxHash: common.RandBytes32(),
		Receiver:   unminted.Receiver,
		Amount:     common.BigIntClone(unminted.Amount),
	}
	err = statedb.UpdateMint(minted)
	assert.NoError(t, err)
	chk, ok, err := statedb.GetMint(minted.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, minted, chk)

	mint := RandMint(true)
	err = statedb.UpdateMint(mint)
	assert.NoError(t, err)
	chk, ok, err = statedb.GetMint(mint.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mint, chk)
}
