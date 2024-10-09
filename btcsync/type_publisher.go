package btcsync

import (
	"sync"

	"github.com/TEENet-io/bridge-go/btcaction"
)

type PublisherService struct {
	DepositObservers         []chan btcaction.DepositAction
	WithdrawObservers        []chan btcaction.WithdrawAction
	UnknownTransferObservers []chan btcaction.UnknownTransferAction
	mu                       sync.Mutex
}

// NewPublisherService creates a new PublisherService
func NewPublisherService() *PublisherService {
	return &PublisherService{
		DepositObservers:         make([]chan btcaction.DepositAction, 0),
		WithdrawObservers:        make([]chan btcaction.WithdrawAction, 0),
		UnknownTransferObservers: make([]chan btcaction.UnknownTransferAction, 0),
	}
}

func (m *PublisherService) RegisterDepositObserver(observer chan btcaction.DepositAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DepositObservers = append(m.DepositObservers, observer)
}

func (m *PublisherService) RegisterWithdrawObserver(observer chan btcaction.WithdrawAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WithdrawObservers = append(m.WithdrawObservers, observer)
}

func (m *PublisherService) RegisterUnknownTransferObserver(observer chan btcaction.UnknownTransferAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UnknownTransferObservers = append(m.UnknownTransferObservers, observer)
}

// // RegisterObserver adds an observer channel to the list
// // observers are classified by the type of action they are interested in
// func (m *PublisherService) RegisterObserver(observer interface{}) error {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()

// 	switch observer := observer.(type) {
// 	case chan btcaction.DepositAction:
// 		m.DepositObservers = append(m.DepositObservers, observer)
// 		return nil
// 	case chan btcaction.WithdrawAction:
// 		m.WithdrawObservers = append(m.WithdrawObservers, observer)
// 		return nil
// 	case chan btcaction.UnknownTransferAction:
// 		m.UnknownTransferObservers = append(m.UnknownTransferObservers, observer)
// 		return nil
// 	default:
// 		return fmt.Errorf("invalid observer type")
// 	}
// }

func (m *PublisherService) NotifyDeposit(da btcaction.DepositAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.DepositObservers {
		select {
		case observer <- da:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan btcaction.DepositAction) {
				obs <- da
			}(observer)
		}
	}
}

func (m *PublisherService) NotifyWithdraw(wa btcaction.WithdrawAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.WithdrawObservers {
		select {
		case observer <- wa:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan btcaction.WithdrawAction) {
				obs <- wa
			}(observer)
		}
	}
}

func (m *PublisherService) NotifyUnknownTransfer(uta btcaction.UnknownTransferAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.UnknownTransferObservers {
		select {
		case observer <- uta:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan btcaction.UnknownTransferAction) {
				obs <- uta
			}(observer)
		}
	}
}
