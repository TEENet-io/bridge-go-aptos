package etherman

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSimulatedChain(t *testing.T) {
	sim := NewSimulatedChain()
	assert.NotNil(t, sim)

	balance, err := sim.Backend.Client().BalanceAt(context.Background(), sim.Accounts[0].From, nil)
	assert.NoError(t, err)
	assert.Equal(t, "100000000000000000000", balance.String())
}
