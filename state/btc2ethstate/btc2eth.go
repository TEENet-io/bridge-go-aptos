package btc2ethstate

import bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"

type State struct {
	mintedEvCh chan *bridge.TEENetBtcBridgeMinted
}

func New() *State {
	return &State{
		mintedEvCh: make(chan *bridge.TEENetBtcBridgeMinted),
	}
}

func (st *State) GetMintedEventChannel() chan *bridge.TEENetBtcBridgeMinted {
	return st.mintedEvCh
}
