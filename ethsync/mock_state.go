package ethsync

import (
	"context"
	"math/big"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
)

const MaxEvNum = 32

type MockEth2BtcState struct {
	newFinalizedCh chan *big.Int
	requestedEvCh  chan *RedeemRequestedEvent
	preparedEvCh   chan *RedeemPreparedEvent

	lastFinalized *big.Int
	requestedEv   []*RedeemRequestedEvent
	preparedEv    []*RedeemPreparedEvent
}

type MockBtc2EthState struct {
	mintedEvCh chan *MintedEvent

	mintedEv []*MintedEvent
}

func NewMockEth2BtcState() *MockEth2BtcState {
	return &MockEth2BtcState{
		newFinalizedCh: make(chan *big.Int, 1),
		requestedEvCh:  make(chan *RedeemRequestedEvent),
		preparedEvCh:   make(chan *RedeemPreparedEvent),

		lastFinalized: new(big.Int).Set(common.EthStartingBlock),
		requestedEv:   make([]*RedeemRequestedEvent, 0, MaxEvNum),
		preparedEv:    make([]*RedeemPreparedEvent, 0, MaxEvNum),
	}
}

func (st *MockEth2BtcState) GetNewRedeemRequestedEventChannel() chan<- *RedeemRequestedEvent {
	return st.requestedEvCh
}

func (st *MockEth2BtcState) GetNewRedeemPreparedEventChannel() chan<- *RedeemPreparedEvent {
	return st.preparedEvCh
}

func (st *MockEth2BtcState) GetFinalizedBlockNumber() (*big.Int, error) {
	return st.lastFinalized, nil
}

func (st *MockEth2BtcState) Start(ctx context.Context) error {
	logger.Info("starting mock eth2btc state")
	defer logger.Info("stopping mock eth2btc state")

	for {
		select {
		case <-ctx.Done():
			return (ctx).Err()
		case n := <-st.newFinalizedCh:
			st.lastFinalized = new(big.Int).Set(n)
		case ev := <-st.requestedEvCh:
			st.requestedEv = append(st.requestedEv, ev)
		case ev := <-st.preparedEvCh:
			st.preparedEv = append(st.preparedEv, ev)
		}
	}
}

func (st *MockEth2BtcState) GetNewFinalizedBlockChannel() chan<- *big.Int {
	return st.newFinalizedCh
}

func NewMockBtc2EthState() *MockBtc2EthState {
	return &MockBtc2EthState{
		mintedEvCh: make(chan *MintedEvent),

		mintedEv: make([]*MintedEvent, 0, MaxEvNum),
	}
}

func (st *MockBtc2EthState) Start(ctx context.Context) error {
	logger.Info("starting mock btc2eth state")
	defer logger.Info("stopping mock btc2eth state")

	for {
		select {
		case <-ctx.Done():
			return (ctx).Err()
		case ev := <-st.mintedEvCh:
			st.mintedEv = append(st.mintedEv, ev)
		}
	}
}

func (st *MockBtc2EthState) GetNewMintedEventChannel() chan<- *MintedEvent {
	return st.mintedEvCh
}
