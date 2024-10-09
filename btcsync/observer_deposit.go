/*
This file implements the deposit action observer.
It will store the deposit action.
*/
package btcsync

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
// You should call it as a separate goroutine
func (s *ObserverDepositAction) GetNotifiedDeposit() {
	for data := range s.Ch {
		s.backend.AddDeposit(data)
	}
}
