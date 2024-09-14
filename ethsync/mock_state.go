package ethsync

import (
	"context"
	"math/big"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/etherman"
)

const MaxEvNum = 32

type MockEth2BtcState struct {
	lastFinalizedCh chan *big.Int
	requestedEvCh   chan *etherman.RedeemRequestedEvent
	preparedEvCh    chan *etherman.RedeemPreparedEvent

	lastFinalized *big.Int
	requestedEv   []*etherman.RedeemRequestedEvent
	preparedEv    []*etherman.RedeemPreparedEvent
}

type MockBtc2EthState struct {
	mintedEvCh chan *etherman.MintedEvent

	mintedEv []*etherman.MintedEvent
}

func NewMockEth2BtcState() *MockEth2BtcState {
	return &MockEth2BtcState{
		lastFinalizedCh: make(chan *big.Int, 1),
		requestedEvCh:   make(chan *etherman.RedeemRequestedEvent),
		preparedEvCh:    make(chan *etherman.RedeemPreparedEvent),

		lastFinalized: big.NewInt(0),
		requestedEv:   make([]*etherman.RedeemRequestedEvent, 0, MaxEvNum),
		preparedEv:    make([]*etherman.RedeemPreparedEvent, 0, MaxEvNum),
	}
}

func (st *MockEth2BtcState) GetRedeemRequestedEventChannel() chan<- *etherman.RedeemRequestedEvent {
	return st.requestedEvCh
}

func (st *MockEth2BtcState) GetRedeemPreparedEventChannel() chan<- *etherman.RedeemPreparedEvent {
	return st.preparedEvCh
}

func (st *MockEth2BtcState) Start(ctx context.Context) error {
	logger.Info("starting mock eth2btc state")
	defer logger.Info("stopping mock eth2btc state")

	for {
		select {
		case <-ctx.Done():
			return (ctx).Err()
		case n := <-st.lastFinalizedCh:
			st.lastFinalized = new(big.Int).Set(n)
		case ev := <-st.requestedEvCh:
			st.requestedEv = append(st.requestedEv, ev)
		case ev := <-st.preparedEvCh:
			st.preparedEv = append(st.preparedEv, ev)
		}
	}
}

func (st *MockEth2BtcState) GetLastEthFinalizedBlockNumberChannel() chan<- *big.Int {
	return st.lastFinalizedCh
}

func NewMockBtc2EthState() *MockBtc2EthState {
	return &MockBtc2EthState{
		mintedEvCh: make(chan *etherman.MintedEvent),

		mintedEv: make([]*etherman.MintedEvent, 0, MaxEvNum),
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

func (st *MockBtc2EthState) GetMintedEventChannel() chan<- *etherman.MintedEvent {
	return st.mintedEvCh
}
