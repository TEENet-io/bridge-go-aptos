package ethsync

import (
	"math/big"
)

type State interface {
	GetNewEthFinalizedBlockChannel() chan<- *big.Int
	GetNewBtcFinalizedBlockChannel() chan<- *big.Int
	GetNewRedeemRequestedEventChannel() chan<- *RedeemRequestedEvent
	GetNewRedeemPreparedEventChannel() chan<- *RedeemPreparedEvent
	GetNewMintedEventChannel() chan<- *MintedEvent

	GetEthFinalizedBlockNumber() (*big.Int, error)
	GetBtcFinalizedBlockNumber() (*big.Int, error)
}
