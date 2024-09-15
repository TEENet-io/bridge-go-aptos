package ethsync

import (
	"math/big"
)

type Eth2BtcState interface {
	GetNewFinalizedBlockChannel() chan<- *big.Int
	GetNewRedeemRequestedEventChannel() chan<- *RedeemRequestedEvent
	GetNewRedeemPreparedEventChannel() chan<- *RedeemPreparedEvent

	GetFinalizedBlockNumber() (*big.Int, error)
}

type Btc2EthState interface {
	GetNewMintedEventChannel() chan<- *MintedEvent
}
