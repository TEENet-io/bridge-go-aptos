package state

import "math/big"

type StateConfig struct {
	ChannelSize   int
	UniqueChainId *big.Int // Eth chain id (eg. 1337), aptos chain id (eg. 1)
}
