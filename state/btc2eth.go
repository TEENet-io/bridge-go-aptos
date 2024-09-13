package state

import bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"

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
