package ethtxmanager

import (
	"database/sql"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestSigReqOps(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	etm, err := NewEthTxManagerDB(db)
	assert.NoError(t, err)
	defer etm.Close()

	sr := &SignatureRequest{
		RequestTxHash: common.RandBytes32(),
		SigningHash:   common.RandBytes32(),
		Outpoints: []state.Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  0,
			},
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			}},
		Rx: big.NewInt(100),
		S:  big.NewInt(200),
	}

	err = etm.InsertSignatureRequest(sr)
	assert.NoError(t, err)

	sr2, ok, err := etm.GetSignatureRequestByRequestTxHash(sr.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, sr, sr2)

	err = etm.RemoveSignatureRequest(sr.RequestTxHash)
	assert.NoError(t, err)

	_, ok, err = etm.GetSignatureRequestByRequestTxHash(sr.RequestTxHash)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestMonitoredTxOps(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	etm, err := NewEthTxManagerDB(db)
	assert.NoError(t, err)
	defer etm.Close()

	mt := &monitoredTx{
		TxHash:    common.RandBytes32(),
		Id:        common.RandBytes32(),
		SentAfter: common.RandBytes32(),
	}

	mts, err := etm.GetMonitoredTxs()
	assert.NoError(t, err)
	assert.Len(t, mts, 0)

	err = etm.InsertMonitoredTx(mt)
	assert.NoError(t, err)

	mt1, ok, err := etm.GetMonitoredTxById(mt.Id)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mt, mt1)

	mts, err = etm.GetMonitoredTxs()
	assert.NoError(t, err)
	assert.Len(t, mts, 1)
	assert.Equal(t, mt, mts[0])

	err = etm.RemoveMonitoredTx(mt.TxHash)
	assert.NoError(t, err)

	_, ok, err = etm.GetMonitoredTxById(mt.Id)
	assert.NoError(t, err)
	assert.False(t, ok)
}
