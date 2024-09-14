package ethsync

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/etherman"
)

type Eth2BtcState interface {
	GetLastEthFinalizedBlockNumberChannel() chan<- *big.Int
	GetRedeemRequestedEventChannel() chan<- *etherman.RedeemRequestedEvent
	GetRedeemPreparedEventChannel() chan<- *etherman.RedeemPreparedEvent

	GetFinalizedBlockNumber() (*big.Int, error)
}

type Btc2EthState interface {
	GetMintedEventChannel() chan<- *etherman.MintedEvent
}
