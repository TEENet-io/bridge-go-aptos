package btcsync

import (
	"sync"

	"github.com/TEENet-io/bridge-go/btcaction"
)

// PublisherService is a concurrent-safe service that
// could "Notify" channels of observers.
// Please "Register" observers via RegisterXXXObserver before Notify.
type PublisherService struct {
	DepositObservers       []chan btcaction.DepositAction
	RedeemObservers        []chan btcaction.RedeemAction
	OtherTransferObservers []chan btcaction.OtherTransferAction
	UTXOObservers          []chan ObservedUTXO
	mu                     sync.Mutex
}

// NewPublisherService creates a new PublisherService
// Currently the observers are empty.
// Add some observers via register.
func NewPublisherService() *PublisherService {
	return &PublisherService{
		DepositObservers:       make([]chan btcaction.DepositAction, 0),
		RedeemObservers:        make([]chan btcaction.RedeemAction, 0),
		OtherTransferObservers: make([]chan btcaction.OtherTransferAction, 0),
		UTXOObservers:          make([]chan ObservedUTXO, 0),
	}
}

// RegisterDepositObserver registers a new observer for deposit action.
func (m *PublisherService) RegisterDepositObserver(observer chan btcaction.DepositAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DepositObservers = append(m.DepositObservers, observer)
}

func (m *PublisherService) RegisterRedeemObserver(observer chan btcaction.RedeemAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RedeemObservers = append(m.RedeemObservers, observer)
}

// RegisterOtherTransferObserver registers a new observer for other transfer action.
func (m *PublisherService) RegisterOtherTransferObserver(observer chan btcaction.OtherTransferAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OtherTransferObservers = append(m.OtherTransferObservers, observer)
}

// RegisterUTXOObserver registers a new observer for UTXO action.
func (m *PublisherService) RegisterUTXOObserver(observer chan ObservedUTXO) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UTXOObservers = append(m.UTXOObservers, observer)
}

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

// Notify "redeem completed" to observers.
func (m *PublisherService) NotifyRedeem(ra btcaction.RedeemAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.RedeemObservers {
		select {
		case observer <- ra:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan btcaction.RedeemAction) {
				obs <- ra
			}(observer)
		}
	}
}

func (m *PublisherService) NotifyOtherTransfer(ota btcaction.OtherTransferAction) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.OtherTransferObservers {
		select {
		case observer <- ota:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan btcaction.OtherTransferAction) {
				obs <- ota
			}(observer)
		}
	}
}

func (m *PublisherService) NotifyUTXO(data ObservedUTXO) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, observer := range m.UTXOObservers {
		select {
		case observer <- data:
		default:
			// Handle the case where the observer's channel is full
			go func(obs chan ObservedUTXO) {
				obs <- data
			}(observer)
		}
	}
}
