package eth2btcstate

import (
	"database/sql"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func getMemoryDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	return db
}
func TestInsertAfterRequested(t *testing.T) {
	sqlDB := getMemoryDB()
	db, err := NewStateDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB.Close()
		db.Close()
	}()

	// Insert a redeem with status == requested
	r0 := randRedeem(RedeemStatusRequested)
	err = db.insertAfterRequested(r0)
	assert.NoError(t, err)
	rs, err := db.GetByStatus(RedeemStatusRequested)
	assert.Equal(t, 1, len(rs))
	r1 := rs[0]
	assert.NoError(t, err)
	assert.Equal(t, r1.RequestTxHash, r0.RequestTxHash)
	assert.Equal(t, r1.Requester, r0.Requester)
	assert.Equal(t, r1.Receiver, r0.Receiver)
	assert.Equal(t, r1.Amount, r0.Amount)
	assert.Equal(t, r1.Status, r0.Status)
	assert.Equal(t, r1.BtcTxId, [32]byte{})
	assert.Equal(t, r1.PrepareTxHash, [32]byte{})
	assert.Nil(t, r1.Outpoints)

	// Cannot insert two redeems with the same request tx hash
	r2 := randRedeem(RedeemStatusRequested)
	r2.Outpoints = nil
	r2.RequestTxHash = r0.RequestTxHash
	err = db.insertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.GetByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rs))
	assert.Equal(t, rs[0], r1)

	// Insert another redeem
	r2.RequestTxHash = common.RandBytes32()
	err = db.insertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.GetByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rs))
	assert.Equal(t, rs[0].RequestTxHash, r0.RequestTxHash)
	assert.Equal(t, rs[1].RequestTxHash, r2.RequestTxHash)
}

func TestUpdateAfterPrepared(t *testing.T) {
	sqlDB := getMemoryDB()
	db, err := NewStateDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB.Close()
		db.Close()
	}()

	// Check errors
	r0 := randRedeem(RedeemStatusPrepared)
	r0.BtcTxId = [32]byte{}
	err = db.updateAfterPrepared(r0)
	assert.NoError(t, err)
	actual, ok, err := db.Get(r0.RequestTxHash[:], RedeemStatusPrepared)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, r0, actual)

	// Update with previous redeem request stored
	r1 := randRedeem(RedeemStatusRequested)
	r1.Outpoints = nil
	r1.BtcTxId = [32]byte{}
	err = db.insertAfterRequested(r1)
	assert.NoError(t, err)
	r1.Status = RedeemStatusPrepared
	r1.Outpoints = []Outpoint{{TxId: common.RandBytes32(), Idx: 0}}
	err = db.updateAfterPrepared(r1)
	assert.NoError(t, err)
	actual, ok, err = db.Get(r1.RequestTxHash[:], RedeemStatusPrepared)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, r1, actual)
}

func TestHas(t *testing.T) {
	sqlDB := getMemoryDB()
	db, err := NewStateDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB.Close()
		db.Close()
	}()

	r := randRedeem(RedeemStatusRequested)

	ok, _, err := db.Has(r.RequestTxHash[:])
	assert.NoError(t, err)
	assert.False(t, ok)

	err = db.insertAfterRequested(r)
	assert.NoError(t, err)
	ok, status, err := db.Has(r.RequestTxHash[:])
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, RedeemStatusRequested, status)
}

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
	err = db.setKeyedValue([]byte("key"), []byte("value1"))
	assert.NoError(t, err)

	// get
	v, err := db.GetKeyedValue([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), ethcommon.TrimLeftZeroes(v))

	err = db.setKeyedValue([]byte("key"), []byte("value2"))
	assert.NoError(t, err)
	v, err = db.GetKeyedValue([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), ethcommon.TrimLeftZeroes(v))
}
