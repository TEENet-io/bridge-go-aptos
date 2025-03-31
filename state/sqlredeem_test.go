package state

import (
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	r0 := RandRedeem(RedeemStatusRequested)
	r1 := &sqlRedeem{}
	var err error

	r0.Outpoints = nil
	r1, err = r1.encode(r0)
	assert.NoError(t, err)
	r2, err := r1.decode()
	assert.NoError(t, err)
	assert.Equal(t, r0, r2)

	r0.Outpoints = []BtcOutpoint{{BtcTxId: common.RandBytes32(), BtcIdx: 0}}
	r1, err = r1.encode(r0)
	assert.NoError(t, err)
	r2, err = r1.decode()
	assert.NoError(t, err)
	assert.Equal(t, r0, r2)
}
