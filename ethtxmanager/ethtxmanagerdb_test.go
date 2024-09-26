package ethtxmanager

import (
	"database/sql"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func newMgr(t *testing.T) (etm *EthTxManagerDB, close func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	etm, err = NewEthTxManagerDB(db)
	assert.NoError(t, err)

	close = func() {
		etm.Close()
		db.Close()
	}

	return etm, close
}

func TestInsertPendingMonitoredTx(t *testing.T) {
	etm, close := newMgr(t)
	defer close()

	mt := RandMonitoredTx(Pending, 1)
	mt.MinedAt = common.RandBytes32()

	err := etm.InsertPendingMonitoredTx(mt)
	assert.NoError(t, err)

	chk, err := etm.GetMonitoredTxsById(mt.Id)
	assert.NoError(t, err)
	assert.Len(t, chk, 1)

	assert.Equal(t, mt.TxHash, chk[0].TxHash)
	assert.Equal(t, mt.Id, chk[0].Id)
	assert.Equal(t, mt.SentAfter, chk[0].SentAfter)

	assert.Equal(t, common.EmptyHash, chk[0].MinedAt)
	assert.Equal(t, Pending, chk[0].Status)
}

func TestGetMonitoredTxsByStatus(t *testing.T) {
	etm, close := newMgr(t)
	defer close()

	mts := []*MonitoredTx{
		RandMonitoredTx(Pending, 1),
		RandMonitoredTx(Pending, 2),
		RandMonitoredTx(Success, 3),
		RandMonitoredTx(Reverted, 4),
	}

	for i, mt := range mts {
		err := etm.InsertPendingMonitoredTx(mt)
		assert.NoError(t, err)

		if i > 1 {
			err := etm.UpdateMonitoredTxAfterMined(mt.TxHash, mt.MinedAt, mt.Status)
			assert.NoError(t, err)
		}
	}

	chk, err := etm.GetMonitoredTxsByStatus(Reorg)
	assert.NoError(t, err)
	assert.Len(t, chk, 0)

	chk, err = etm.GetMonitoredTxsByStatus(Pending)
	assert.NoError(t, err)
	assert.Len(t, chk, 2)
	assert.Equal(t, mts[0], chk[0])
	assert.Equal(t, mts[1], chk[1])

	chk, err = etm.GetMonitoredTxsByStatus(Success)
	assert.NoError(t, err)
	assert.Len(t, chk, 1)
	assert.Equal(t, mts[2], chk[0])
}

func TestUpdateMonitoredTxStatus(t *testing.T) {
	etm, close := newMgr(t)
	defer close()

	mt := RandMonitoredTx(Pending, 1)
	err := etm.InsertPendingMonitoredTx(mt)
	assert.NoError(t, err)

	err = etm.UpdateMonitoredTxStatus(mt.TxHash, "Invalid_Status")
	assert.Error(t, err)

	err = etm.UpdateMonitoredTxStatus(mt.TxHash, Success)
	assert.NoError(t, err)

	chk, err := etm.GetMonitoredTxsById(mt.Id)
	assert.NoError(t, err)
	mt.Status = Success
	assert.Equal(t, mt, chk[0])
}

func TestUpdateMonitoredTxAfterMined(t *testing.T) {
	etm, close := newMgr(t)
	defer close()

	mt := RandMonitoredTx(Pending, 1)
	err := etm.InsertPendingMonitoredTx(mt)
	assert.NoError(t, err)

	minedAt := common.RandBytes32()
	mt.MinedAt = minedAt
	mt.Status = Success
	err = etm.UpdateMonitoredTxAfterMined(mt.TxHash, minedAt, Success)
	assert.NoError(t, err)

	chk, err := etm.GetMonitoredTxsById(mt.Id)
	assert.NoError(t, err)
	assert.Len(t, chk, 1)
	assert.Equal(t, mt, chk[0])
}

func TestGetMonitoredTxsById(t *testing.T) {
	etm, close := newMgr(t)
	defer close()

	mts := []*MonitoredTx{
		RandMonitoredTx(Pending, 1),
		RandMonitoredTx(Timeout, 2),
	}
	mts[1].Id = mts[0].Id
	for _, mt := range mts {
		err := etm.InsertMonitoredTx(mt)
		assert.NoError(t, err)
	}

	chk, err := etm.GetMonitoredTxsById(mts[0].Id)
	assert.NoError(t, err)
	assert.Len(t, chk, 2)
	assert.Equal(t, mts[0], chk[0])
	assert.Equal(t, mts[1], chk[1])
}
