package btcwallet

import (
	"database/sql"
	"math/big"
	"os"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func randFile() string {
	return "./" + ethcommon.Hash(common.RandBytes32()).String() + ".db"
}

func newBtcWalletDB(t *testing.T) (*BtcWalletDB, func()) {
	file := randFile()
	db, err := sql.Open("sqlite3", file)
	assert.NoError(t, err)

	btcWalletDB, err := NewBtcWalletDB(db)
	assert.NoError(t, err)

	close := func() {
		btcWalletDB.Close()
		db.Close()
		os.Remove(file)
	}

	return btcWalletDB, close
}

func TestInsert(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, true)
	err := btcWalletDB.Insert(spendable)
	assert.NoError(t, err)

	chk, ok, err := btcWalletDB.GetById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, spendable, chk)
}

func TestLock(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, false)
	err := btcWalletDB.Insert(spendable)
	assert.NoError(t, err)

	err = btcWalletDB.SetLock(spendable.BtcTxId, true)
	assert.NoError(t, err)
	chk, ok, err := btcWalletDB.GetById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, chk.Lock)

	err = btcWalletDB.SetLock(spendable.BtcTxId, false)
	assert.NoError(t, err)
	chk, ok, err = btcWalletDB.GetById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.False(t, chk.Lock)
}

func TestDelete(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, false)
	err := btcWalletDB.Insert(spendable)
	assert.NoError(t, err)

	err = btcWalletDB.Delete(spendable.BtcTxId)
	assert.NoError(t, err)

	_, ok, err := btcWalletDB.GetById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestGetSpendables(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	dat := []*Spendable{
		RandSpendable(100, 1, false),
		RandSpendable(150, 1, false),
		RandSpendable(50, 2, false),
		RandSpendable(200, 2, false),
		RandSpendable(300, 2, false),
		RandSpendable(10000, 1, true),
	}
	for _, spendable := range dat {
		err := btcWalletDB.Insert(spendable)
		assert.NoError(t, err)
	}

	// only spendables in block 1
	amount := big.NewInt(240)
	spendables, ok, err := btcWalletDB.GetSpendables(amount)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, spendables, 2)
	for i := 0; i < len(spendables); i++ {
		assert.Equal(t, dat[i], spendables[i])
	}

	// spenables in block 1 + first in block 2
	amount = big.NewInt(300)
	spendables, ok, err = btcWalletDB.GetSpendables(amount)
	assert.NoError(t, err)
	assert.True(t, ok)
	for i := 0; i < len(spendables); i++ {
		assert.Equal(t, dat[i], spendables[i])
	}

	// fail to get spendables
	amount = big.NewInt(801)
	spendables, ok, err = btcWalletDB.GetSpendables(amount)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Len(t, spendables, 0)
}
