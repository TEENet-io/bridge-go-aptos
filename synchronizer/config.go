package synchronizer

import (
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
)

type EthSyncConfig struct {
	Etherman                     *etherman.Etherman
	CheckFinalizedTickerInterval time.Duration
	Btc2EthState                 state.Btc2EthState
	Eth2BtcState                 state.Eth2BtcState
	LastFinalizedBLock           *big.Int
}
