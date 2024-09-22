package ethtxmanager

import (
	"database/sql"
	"math/big"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	_ "github.com/mattn/go-sqlite3"
)

func TestNewEthTxManagerDB(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	etm, err := NewEthTxManagerDB(db)
	assert.NoError(t, err)
	defer etm.Close()
}

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
		Rx:            big.NewInt(100),
		S:             big.NewInt(200),
	}

	err = etm.insertSignatureRequest(sr)
	assert.NoError(t, err)

	sr2, ok, err := etm.GetSignatureRequestByRequestTxHash(sr.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, sr, sr2)
}

func TestMonitoredTxOps(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)
	defer db.Close()

	etm, err := NewEthTxManagerDB(db)
	assert.NoError(t, err)
	defer etm.Close()

	mt := &monitoredTx{
		TxHash:        common.RandBytes32(),
		RequestTxHash: common.RandBytes32(),
		SentAfter:     common.RandBytes32(),
		MinedAt:       common.RandBytes32(),
		Status:        MonitoredTxStatusSuccess,
	}

	err = etm.insertMonitoredTx(mt)
	assert.NoError(t, err)
	mt1, ok, err := etm.GetMonitoredTxByRequestTxHash(mt.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mt1.MinedAt, ethcommon.BytesToHash([]byte{}))
	assert.Equal(t, mt1.Status, MonitoredTxStatusPending)
	mt1.MinedAt = mt.MinedAt
	mt1.Status = mt.Status
	assert.Equal(t, mt, mt1)

	err = etm.updateMonitoredTxAfterMined(mt)
	assert.NoError(t, err)
	mt2, ok, err := etm.GetMonitoredTxByRequestTxHash(mt.RequestTxHash)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, mt, mt2)

	mt.MinedAt = [32]byte{}
	err = etm.updateMonitoredTxAfterMined(mt)
	assert.Equal(t, ErrMintedAtNotSet, err)
}
