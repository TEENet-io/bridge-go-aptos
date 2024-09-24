package state

import (
	"testing"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	r0 := randRedeem(RedeemStatusRequested)

	r0.Outpoints = nil
	r1, err := encode(r0)
	assert.NoError(t, err)
	r2, err := r1.decode()
	assert.NoError(t, err)
	assert.Equal(t, r0, r2)

	r0.Outpoints = []Outpoint{{TxId: common.RandBytes32(), Idx: 0}}
	r1, err = encode(r0)
	assert.NoError(t, err)
	r2, err = r1.decode()
	assert.NoError(t, err)
	assert.Equal(t, r0, r2)
}
