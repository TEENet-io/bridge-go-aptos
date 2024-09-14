package ethsync

import (
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/etherman"
)

type Config struct {
	Etherman                     *etherman.Etherman
	CheckFinalizedTickerInterval time.Duration
	Btc2EthState                 Btc2EthState
	Eth2BtcState                 Eth2BtcState
	LastFinalizedBLock           *big.Int
}
