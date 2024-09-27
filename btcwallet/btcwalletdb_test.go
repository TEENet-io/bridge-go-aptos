package btcwallet

import (
	"database/sql"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
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

func TestInsertRequest(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	req := &Request{
		Id: common.RandBytes32(),
		Outpoints: []state.Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			},
		},
		Status: Timeout,
	}
	err := btcWalletDB.InsertRequest(req)
	assert.NoError(t, err)

	chk, ok, err := btcWalletDB.GetRequestById(req.Id)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, req.Id, chk.Id)
	assert.Equal(t, req.Outpoints, chk.Outpoints)
	assert.Equal(t, req.Status, chk.Status)
	assert.Less(t, chk.CreatedAt, time.Now())
}

func TestUpdateRequestStatus(t *testing.T) {
	db, close := newBtcWalletDB(t)
	defer close()

	req := &Request{
		Id: common.RandBytes32(),
		Outpoints: []state.Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			},
		},
		Status: Locked,
	}
	err := db.InsertRequest(req)
	assert.NoError(t, err)

	err = db.UpdateRequestStatus(req.Id, Timeout)
	assert.NoError(t, err)

	chk, ok, err := db.GetRequestById(req.Id)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, Timeout, chk.Status)
}

func TestGetRequestsByStatus(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	reqs := []*Request{
		{
			Id: common.RandBytes32(),
			Outpoints: []state.Outpoint{
				{
					TxId: common.RandBytes32(),
					Idx:  1,
				},
			},
			Status: Locked,
		},
		{
			Id: common.RandBytes32(),
			Outpoints: []state.Outpoint{
				{
					TxId: common.RandBytes32(),
					Idx:  1,
				},
			},
			Status: Timeout,
		},
	}
	for _, req := range reqs {
		err := btcWalletDB.InsertRequest(req)
		assert.NoError(t, err)
	}

	requests, err := btcWalletDB.GetRequestsByStatus(Locked)
	assert.NoError(t, err)
	assert.Len(t, requests, 1)
	reqs[0].CreatedAt = requests[0].CreatedAt
	assert.Equal(t, reqs[0], requests[0])
}

func TestDeleteRequest(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	req := &Request{
		Id: common.RandBytes32(),
		Outpoints: []state.Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			},
		},
		Status: Locked,
	}
	err := btcWalletDB.InsertRequest(req)
	assert.NoError(t, err)

	err = btcWalletDB.DeleteRequest(req.Id)
	assert.NoError(t, err)

	_, ok, err := btcWalletDB.GetRequestById(req.Id)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestInsertSpendable(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, true)
	err := btcWalletDB.InsertSpendable(spendable)
	assert.NoError(t, err)

	chk, ok, err := btcWalletDB.GetSpendableById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, spendable, chk)
}

func TestLock(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, false)
	err := btcWalletDB.InsertSpendable(spendable)
	assert.NoError(t, err)

	err = btcWalletDB.SetLockOnSpendable(spendable.BtcTxId, true)
	assert.NoError(t, err)
	chk, ok, err := btcWalletDB.GetSpendableById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, chk.Lock)

	err = btcWalletDB.SetLockOnSpendable(spendable.BtcTxId, false)
	assert.NoError(t, err)
	chk, ok, err = btcWalletDB.GetSpendableById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.False(t, chk.Lock)
}

func TestDelete(t *testing.T) {
	btcWalletDB, close := newBtcWalletDB(t)
	defer close()

	spendable := RandSpendable(100, 1, false)
	err := btcWalletDB.InsertSpendable(spendable)
	assert.NoError(t, err)

	err = btcWalletDB.DeleteSpendable(spendable.BtcTxId)
	assert.NoError(t, err)

	_, ok, err := btcWalletDB.GetSpendableById(spendable.BtcTxId)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestRequestSpendablesByAmount(t *testing.T) {
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
		err := btcWalletDB.InsertSpendable(spendable)
		assert.NoError(t, err)
	}

	// only spendables in block 1
	amount := big.NewInt(240)
	spendables, ok, err := btcWalletDB.RequestSpendablesByAmount(amount)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, spendables, 2)
	for i := 0; i < len(spendables); i++ {
		assert.Equal(t, dat[i], spendables[i])
	}

	// spenables in block 1 + first in block 2
	amount = big.NewInt(300)
	spendables, ok, err = btcWalletDB.RequestSpendablesByAmount(amount)
	assert.NoError(t, err)
	assert.True(t, ok)
	for i := 0; i < len(spendables); i++ {
		assert.Equal(t, dat[i], spendables[i])
	}

	// fail to get spendables
	amount = big.NewInt(801)
	spendables, ok, err = btcWalletDB.RequestSpendablesByAmount(amount)
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Len(t, spendables, 0)
}
