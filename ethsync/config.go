package ethsync

import (
	"time"

	"github.com/TEENet-io/bridge-go/etherman"
)

type Config struct {
	Etherman                          *etherman.Etherman
	CheckLatestFinalizedBlockInterval time.Duration
	Btc2EthState                      Btc2EthState
	Eth2BtcState                      Eth2BtcState
}
