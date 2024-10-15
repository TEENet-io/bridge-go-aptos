package btcsync

/*
This file implements the DepositAction observer.
It stores the DepositAction into backend.
*/

import (
	"github.com/TEENet-io/bridge-go/btcaction"
)

type ObserverDepositAction struct {
	backend btcaction.DepositStorage
	Ch      chan btcaction.DepositAction // communication channel
}

func NewObserverDepositAction(backend btcaction.DepositStorage, bufferSize int) *ObserverDepositAction {
	return &ObserverDepositAction{
		backend: backend,
		Ch:      make(chan btcaction.DepositAction, bufferSize),
	}
}

// GetNotifiedDeposit implements the DepositObserver interface
// You should init it as a separate goroutine (with go)
func (s *ObserverDepositAction) GetNotifiedDeposit() {
	for data := range s.Ch {
		s.backend.AddDeposit(data)
	}
}
