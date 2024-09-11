package state

import (
	"math/big"

	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
)

type Eth2BtcStateConfig struct {
	LastFinalizedBLock uint32
}

type Eth2BtcState interface {
	GetLastEthFinalizedBlockNumberChannel() chan *big.Int
	GetRedeemRequestedEventChannel() chan *bridge.TEENetBtcBridgeRedeemRequested
	GetRedeemPreparedEventChannel() chan *bridge.TEENetBtcBridgeRedeemPrepared
}

type Btc2EthState interface {
	GetMintedEventChannel() chan *bridge.TEENetBtcBridgeMinted
}
