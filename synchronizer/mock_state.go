package synchronizer

import (
	"math/big"

	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
)

const MaxEvNum = 32

type MockEth2BtcState struct {
	lastFinalizedCh chan *big.Int
	requestedEvCh   chan *bridge.TEENetBtcBridgeRedeemRequested
	preparedEvCh    chan *bridge.TEENetBtcBridgeRedeemPrepared

	lastFinalized *big.Int
	requestedEv   []*bridge.TEENetBtcBridgeRedeemRequested
	preparedEv    []*bridge.TEENetBtcBridgeRedeemPrepared
}

type MockBtc2EthState struct {
	mintedEvCh chan *bridge.TEENetBtcBridgeMinted

	mintedEv []*bridge.TEENetBtcBridgeMinted
}

func NewMockEth2BtcState() *MockEth2BtcState {
	return &MockEth2BtcState{
		lastFinalizedCh: make(chan *big.Int, 1),
		requestedEvCh:   make(chan *bridge.TEENetBtcBridgeRedeemRequested, MaxEvNum),
		preparedEvCh:    make(chan *bridge.TEENetBtcBridgeRedeemPrepared, MaxEvNum),

		lastFinalized: big.NewInt(0),
		requestedEv:   make([]*bridge.TEENetBtcBridgeRedeemRequested, 0, MaxEvNum),
		preparedEv:    make([]*bridge.TEENetBtcBridgeRedeemPrepared, 0, MaxEvNum),
	}
}

func (st *MockEth2BtcState) GetRedeemRequestedEventChannel() chan *bridge.TEENetBtcBridgeRedeemRequested {
	return st.requestedEvCh
}

func (st *MockEth2BtcState) GetRedeemPreparedEventChannel() chan *bridge.TEENetBtcBridgeRedeemPrepared {
	return st.preparedEvCh
}

func (st *MockEth2BtcState) GetLastEthFinalizedBlockNumberChannel() chan *big.Int {
	return st.lastFinalizedCh
}

func NewMockBtc2EthState() *MockBtc2EthState {
	return &MockBtc2EthState{
		mintedEvCh: make(chan *bridge.TEENetBtcBridgeMinted, MaxEvNum),

		mintedEv: make([]*bridge.TEENetBtcBridgeMinted, 0, MaxEvNum),
	}
}

func (st *MockBtc2EthState) GetMintedEventChannel() chan *bridge.TEENetBtcBridgeMinted {
	return st.mintedEvCh
}
