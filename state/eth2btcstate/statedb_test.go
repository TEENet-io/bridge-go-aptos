package eth2btcstate

import (
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestInsertAfterRequested(t *testing.T) {
	db, err := newStateDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Insert a redeem with status == requested
	r0 := randRedeem(RedeemStatusRequested)
	err = db.insertAfterRequested(r0)
	assert.NoError(t, err)
	rs, err := db.getByStatus(RedeemStatusRequested)
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

	// Can only insert when status == requested or invalid
	r0.Status = RedeemStatusPrepared
	err = db.insertAfterRequested(r0)
	assert.Equal(t, err, stateDBErrors.CannotInsertDueToInvalidStatus(r0))

	// Cannot insert two redeems with the same request tx hash
	r2 := randRedeem(RedeemStatusRequested)
	r2.Outpoints = nil
	r2.RequestTxHash = r0.RequestTxHash
	err = db.insertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.getByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rs))
	assert.Equal(t, rs[0], r1)

	// Insert another redeem
	r2.RequestTxHash = common.RandBytes32()
	err = db.insertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.getByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(rs))
	assert.Equal(t, rs[0].RequestTxHash, r0.RequestTxHash)
	assert.Equal(t, rs[1].RequestTxHash, r2.RequestTxHash)
}

func TestUpdateAfterPrepared(t *testing.T) {
	db, err := newStateDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Check errors
	r0 := randRedeem(RedeemStatusRequested)
	err = db.updateAfterPrepared(r0)
	assert.Equal(t, err, stateDBErrors.CannotUpdateDueToInvalidStatus(r0))

	// Insert a redeem with status == requested
	r1 := randRedeem(RedeemStatusRequested)
	r1.Outpoints = nil
	err = db.insertAfterRequested(r1)
	assert.NoError(t, err)

	// update
	r1.Status = RedeemStatusPrepared
	r1.Outpoints = []Outpoint{{TxId: common.RandBytes32(), Idx: 0}}
	err = db.updateAfterPrepared(r1)
	assert.NoError(t, err)

	// check
	rs, err := db.getByStatus(RedeemStatusPrepared)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rs))
	r2 := rs[0]
	assert.Equal(t, r2.BtcTxId, [32]byte{})
}

func TestHas(t *testing.T) {
	db, err := newStateDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	r := randRedeem(RedeemStatusRequested)

	ok, _, err := db.has(r.RequestTxHash[:])
	assert.NoError(t, err)
	assert.False(t, ok)

	err = db.insertAfterRequested(r)
	assert.NoError(t, err)
	ok, status, err := db.has(r.RequestTxHash[:])
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, RedeemStatusRequested, status)
}

func TestKV(t *testing.T) {
	db, err := newStateDB("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// insert
	err = db.KVSet([]byte("key"), []byte("value1"))
	assert.NoError(t, err)

	// get
	v, err := db.KVGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), ethcommon.TrimLeftZeroes(v))

	err = db.KVSet([]byte("key"), []byte("value2"))
	assert.NoError(t, err)
	v, err = db.KVGet([]byte("key"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value2"), ethcommon.TrimLeftZeroes(v))
}
