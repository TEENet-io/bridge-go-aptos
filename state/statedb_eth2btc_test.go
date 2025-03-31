package state

import (
	"log"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func newTestStateDBEnv(t *testing.T) (*StateDB, func()) {
	sqlDB := getMemoryDB()
	statedb, err := NewStateDB(sqlDB)
	assert.NoError(t, err)
	return statedb, func() {
		statedb.Close()
		sqlDB.Close()
	}
}

func TestGetRedeemWithNull(t *testing.T) {
	db, close := newTestStateDBEnv(t)
	defer close()

	expected := RandRedeem(RedeemStatusRequested)
	err := db.InsertAfterRequested(expected)
	assert.NoError(t, err)
	actual, ok, err := db.GetRedeem(expected.RequestTxHash)
	if err != nil {
		log.Fatal(err)
	}

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, expected.RequestTxHash, actual.RequestTxHash)
	assert.Equal(t, expected.Requester, actual.Requester)
	assert.Equal(t, expected.Receiver, actual.Receiver)
	assert.Equal(t, expected.Amount, actual.Amount)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, ethcommon.Hash{}, actual.BtcTxId)
	assert.Equal(t, ethcommon.Hash{}, actual.PrepareTxHash)
	assert.Len(t, actual.Outpoints, 0)
}

func TestGetRedeemsByStatus(t *testing.T) {
	db, close := newTestStateDBEnv(t)
	defer close()

	expected := []*Redeem{
		RandRedeem(RedeemStatusRequested),
		RandRedeem(RedeemStatusRequested),
	}

	for _, redeem := range expected {
		err := db.InsertAfterRequested(redeem)
		assert.NoError(t, err)
	}

	actual, err := db.GetRedeemsByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Len(t, actual, len(expected))
	for i := range expected {
		assert.Equal(t, expected[i].RequestTxHash, actual[i].RequestTxHash)
		assert.Equal(t, expected[i].Requester, actual[i].Requester)
		assert.Equal(t, expected[i].Receiver, actual[i].Receiver)
		assert.Equal(t, expected[i].Amount, actual[i].Amount)
		assert.Equal(t, expected[i].Status, actual[i].Status)
		assert.Equal(t, ethcommon.Hash{}, actual[i].BtcTxId)
		assert.Equal(t, ethcommon.Hash{}, actual[i].PrepareTxHash)
		assert.Len(t, actual[i].Outpoints, 0)
	}
}

func TestInsertAfterRequested(t *testing.T) {
	db, close := newTestStateDBEnv(t)
	defer close()

	// Insert a redeem with status == requested
	r0 := RandRedeem(RedeemStatusRequested)
	err := db.InsertAfterRequested(r0)
	assert.NoError(t, err)
	rs, err := db.GetRedeemsByStatus(RedeemStatusRequested)
	assert.Equal(t, 1, len(rs))
	r1 := rs[0]
	assert.NoError(t, err)
	assert.Equal(t, r1.RequestTxHash, r0.RequestTxHash)
	assert.Equal(t, r1.Requester, r0.Requester)
	assert.Equal(t, r1.Receiver, r0.Receiver)
	assert.Equal(t, r1.Amount, r0.Amount)
	assert.Equal(t, r1.Status, r0.Status)
	assert.Equal(t, r1.BtcTxId, ethcommon.Hash{})
	assert.Equal(t, r1.PrepareTxHash, ethcommon.Hash{})
	assert.Nil(t, r1.Outpoints)

	// Cannot insert two redeems with the same request tx hash
	r2 := RandRedeem(RedeemStatusRequested)
	r2.Outpoints = nil
	r2.RequestTxHash = r0.RequestTxHash
	err = db.InsertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.GetRedeemsByStatus(RedeemStatusRequested)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rs))
	assert.Equal(t, rs[0], r1)

	// Insert another redeem
	r2.RequestTxHash = common.RandBytes32()
	err = db.InsertAfterRequested(r2)
	assert.NoError(t, err)
	rs, err = db.GetRedeemsByStatus(RedeemStatusRequested)
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
	r0 := RandRedeem(RedeemStatusPrepared)
	r0.BtcTxId = [32]byte{}
	err = db.UpdateAfterPrepared(r0)
	assert.NoError(t, err)
	actual, ok, err := db.GetRedeem(r0.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, r0, actual)

	// Update with previous redeem request stored
	r1 := RandRedeem(RedeemStatusRequested)
	r1.Outpoints = nil
	r1.BtcTxId = [32]byte{}
	err = db.InsertAfterRequested(r1)
	assert.NoError(t, err)
	r1.Status = RedeemStatusPrepared
	r1.Outpoints = []BtcOutpoint{{BtcTxId: common.RandBytes32(), BtcIdx: 0}}
	err = db.UpdateAfterPrepared(r1)
	assert.NoError(t, err)
	actual, ok, err = db.GetRedeem(r1.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, r1, actual)
}

func TestHasRedeem(t *testing.T) {
	sqlDB := getMemoryDB()
	db, err := NewStateDB(sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB.Close()
		db.Close()
	}()

	r := RandRedeem(RedeemStatusRequested)

	ok, _, err := db.HasRedeem(r.RequestTxHash)
	assert.NoError(t, err)
	assert.False(t, ok)

	err = db.InsertAfterRequested(r)
	assert.NoError(t, err)
	ok, status, err := db.HasRedeem(r.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, RedeemStatusRequested, status)
}
