/*
This file implements the deposit action observer.
It will store the depositaction.
*/
package btcsync

import (
	"github.com/TEENet-io/bridge-go/btcaction"
)

type DepositStorageService struct {
	backend btcaction.DepositStorage
	Ch      chan btcaction.DepositAction // communication channel
}

func NewDepositStorageSerivice(backend btcaction.DepositStorage, bufferSize int) *DepositStorageService {
	return &DepositStorageService{
		backend: backend,
		Ch:      make(chan btcaction.DepositAction, bufferSize),
	}
}

// GetNotifiedDeposit implements the DepositObserver interface
// You should call it as a separate goroutine
func (s *DepositStorageService) GetNotifiedDeposit(da btcaction.DepositAction) {
	for b := range s.Ch {
		s.backend.AddDeposit(b)
	}
}
