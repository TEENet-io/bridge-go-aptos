package state

import "math/big"

type Config struct {
	ChannelSize int
	EthChainId  *big.Int
}
