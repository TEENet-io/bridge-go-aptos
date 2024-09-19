package btc2ethstate

import (
	"github.com/TEENet-io/bridge-go/ethsync"
)

type State struct {
	mintedEvCh chan *ethsync.MintedEvent
}

func New() *State {
	return &State{
		mintedEvCh: make(chan *ethsync.MintedEvent),
	}
}

func (st *State) Close() {
	// to be implemented
}

func (st *State) GetNewMintedEventChannel() chan<- *ethsync.MintedEvent {
	return st.mintedEvCh
}
