package etherman

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ETH_ACCOUNTS = 10

func TestNewSimulatedChain(t *testing.T) {
	sim := NewSimulatedChain(GenPrivateKeys(ETH_ACCOUNTS), big.NewInt(1337))
	assert.NotNil(t, sim)

	balance, err := sim.Backend.Client().BalanceAt(context.Background(), sim.Accounts[0].From, nil)
	assert.NoError(t, err)
	assert.Equal(t, "100000000000000000000", balance.String())
}
