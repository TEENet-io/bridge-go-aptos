package aptossync

import (
	"context"
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	logger "github.com/sirupsen/logrus"
)

const MaxEvNum = 32

type MockState struct {
	newAptosFinalizVersionCh chan *big.Int
	newBtcFinalizedCh        chan *big.Int
	mintedEvCh               chan *agreement.MintedEvent
	requestedEvCh            chan *agreement.RedeemRequestedEvent
	preparedEvCh             chan *agreement.RedeemPreparedEvent

	lastAptosFinalizVersion *big.Int
	lastBtcFinalized        *big.Int

	mintedEv    []*agreement.MintedEvent
	requestedEv []*agreement.RedeemRequestedEvent
	preparedEv  []*agreement.RedeemPreparedEvent
}

func NewMockState() *MockState {
	return &MockState{
		newAptosFinalizVersionCh: make(chan *big.Int, 1),
		newBtcFinalizedCh:        make(chan *big.Int, 1),
		mintedEvCh:               make(chan *agreement.MintedEvent, 1),
		requestedEvCh:            make(chan *agreement.RedeemRequestedEvent, 1),
		preparedEvCh:             make(chan *agreement.RedeemPreparedEvent, 1),

		lastAptosFinalizVersion: new(big.Int).Set(common.EthStartingBlock),
		lastBtcFinalized:        big.NewInt(0),
		mintedEv:                []*agreement.MintedEvent{},
		requestedEv:             []*agreement.RedeemRequestedEvent{},
		preparedEv:              []*agreement.RedeemPreparedEvent{},
	}
}

func (st *MockState) GetNewBlockChainFinalizedLedgerNumberChannel() chan<- *big.Int {
	return st.newAptosFinalizVersionCh
}

func (st *MockState) GetNewBtcFinalizedBlockChannel() chan<- *big.Int {
	return st.newBtcFinalizedCh
}
func (st *MockState) GetNewMintedEventChannel() chan<- *agreement.MintedEvent {
	return st.mintedEvCh
}

func (st *MockState) GetNewRedeemRequestedEventChannel() chan<- *agreement.RedeemRequestedEvent {
	return st.requestedEvCh
}

func (st *MockState) GetNewRedeemPreparedEventChannel() chan<- *agreement.RedeemPreparedEvent {
	return st.preparedEvCh
}

func (st *MockState) GetBlockchainFinalizedBlockNumber() (*big.Int, error) {
	return st.lastAptosFinalizVersion, nil
}

func (st *MockState) Start(ctx context.Context) error {
	logger.Debug("starting mock state")
	defer logger.Debug("stopping mock state")

	for {
		select {
		case <-ctx.Done():
			return (ctx).Err()
		case n := <-st.newAptosFinalizVersionCh:
			logger.Debugf("new aptos finalized version: %v", n)
			st.lastAptosFinalizVersion = new(big.Int).Set(n)
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
