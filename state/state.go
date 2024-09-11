package state

import (
	"math/big"

	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
)

var (
	MaxPendingRequestedEv = 1024
	MaxPendingPreparedEv  = 1024
)

type E2BState struct {
	lastFinalizedCh chan *big.Int
	requestedEvCh   chan *bridge.TEENetBtcBridgeRedeemRequested
	preparedEvCh    chan *bridge.TEENetBtcBridgeRedeemPrepared
}

func NewEth2BtcState() *E2BState {
	return &E2BState{
		lastFinalizedCh: make(chan *big.Int, 1),
		requestedEvCh:   make(chan *bridge.TEENetBtcBridgeRedeemRequested, MaxPendingRequestedEv),
		preparedEvCh:    make(chan *bridge.TEENetBtcBridgeRedeemPrepared, MaxPendingPreparedEv),
	}
}

func (st *E2BState) GetLastEthFinalizedBlockNumberChannel() chan *big.Int {
	return st.lastFinalizedCh
}

func (st *E2BState) GetRedeemRequestedEventChannel() chan *bridge.TEENetBtcBridgeRedeemRequested {
	return st.requestedEvCh
}

func (st *E2BState) GetRedeemPreparedEventChannel() chan *bridge.TEENetBtcBridgeRedeemPrepared {
	return st.preparedEvCh
}

type B2EState struct {
	mintedEvCh chan *bridge.TEENetBtcBridgeMinted
}

func NewBtc2EthState() *B2EState {
	return &B2EState{
		mintedEvCh: make(chan *bridge.TEENetBtcBridgeMinted, MaxPendingRequestedEv),
	}
}

func (st *B2EState) GetMintedEventChannel() chan *bridge.TEENetBtcBridgeMinted {
	return st.mintedEvCh
}
