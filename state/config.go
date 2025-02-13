package state

import "math/big"

type StateConfig struct {
	ChannelSize int
	EthChainId  *big.Int
}
