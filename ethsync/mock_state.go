package ethsync

import (
	"context"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	logger "github.com/sirupsen/logrus"
)

const MaxEvNum = 32

type MockState struct {
	newEthFinalizedCh chan *big.Int
	newBtcFinalizedCh chan *big.Int
	mintedEvCh        chan *MintedEvent
	requestedEvCh     chan *RedeemRequestedEvent
	preparedEvCh      chan *RedeemPreparedEvent

	lastEthFinalized *big.Int
	lastBtcFinalized *big.Int

	mintedEv    []*MintedEvent
	requestedEv []*RedeemRequestedEvent
	preparedEv  []*RedeemPreparedEvent
}

func NewMockState() *MockState {
	return &MockState{
		newEthFinalizedCh: make(chan *big.Int, 1),
		newBtcFinalizedCh: make(chan *big.Int, 1),
		mintedEvCh:        make(chan *MintedEvent, 1),
		requestedEvCh:     make(chan *RedeemRequestedEvent, 1),
		preparedEvCh:      make(chan *RedeemPreparedEvent, 1),

		lastEthFinalized: new(big.Int).Set(common.EthStartingBlock),
		lastBtcFinalized: big.NewInt(0),
		mintedEv:         []*MintedEvent{},
		requestedEv:      []*RedeemRequestedEvent{},
		preparedEv:       []*RedeemPreparedEvent{},
	}
}

func (st *MockState) GetNewEthFinalizedBlockChannel() chan<- *big.Int {
	return st.newEthFinalizedCh
}

func (st *MockState) GetNewBtcFinalizedBlockChannel() chan<- *big.Int {
	return st.newBtcFinalizedCh
}
func (st *MockState) GetNewMintedEventChannel() chan<- *MintedEvent {
	return st.mintedEvCh
}

func (st *MockState) GetNewRedeemRequestedEventChannel() chan<- *RedeemRequestedEvent {
	return st.requestedEvCh
}

func (st *MockState) GetNewRedeemPreparedEventChannel() chan<- *RedeemPreparedEvent {
	return st.preparedEvCh
}

func (st *MockState) GetEthFinalizedBlockNumber() (*big.Int, error) {
	return st.lastEthFinalized, nil
}

func (st *MockState) Start(ctx context.Context) error {
	logger.Debug("starting mock state")
	defer logger.Debug("stopping mock state")

	for {
		select {
		case <-ctx.Done():
			return (ctx).Err()
		case n := <-st.newEthFinalizedCh:
			logger.Debugf("new eth finalized block: %v", n)
			st.lastEthFinalized = new(big.Int).Set(n)
		case ev := <-st.mintedEvCh:
			logger.Debugf("new minted event: %v", ev)
			st.mintedEv = append(st.mintedEv, ev)
		case ev := <-st.requestedEvCh:
			logger.Debugf("new requested event: %v", ev)
			st.requestedEv = append(st.requestedEv, ev)
		case ev := <-st.preparedEvCh:
			logger.Debugf("new prepared event: %v", ev)
			st.preparedEv = append(st.preparedEv, ev)
		}
	}
}
