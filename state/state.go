package state

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	KeyEthFinalizedBlock = crypto.Keccak256Hash([]byte("KeyEthFinalizedBlock"))
	KeyBtcFinalizedBlock = crypto.Keccak256Hash([]byte("KeyBtcFinalizedBlock"))

	ErrSetEthFinalizedBlockNumber           = errors.New("failed to set eth finalized block number in statedb")
	ErrGetEthFinalizedBlockNumber           = errors.New("failed to get eth finalized block number from statedb")
	ErrStoredEthFinalizedBlockNumberInvalid = errors.New("stored eth finalized block number is invalid")

	ErrRedeemInvalid          = errors.New("redeem is invalid")
	ErrGetRedeem              = errors.New("failed to get redeem from statedb")
	ErrCheckRedeemExistence   = errors.New("failed to check redeem existence")
	ErrPreparedEventUnmatched = errors.New("redeem prepared event is unmatched with stored requested redeem")
	ErrUpdateRedeem           = errors.New("failed to update redeem in statedb")
	ErrPreparedEventInvalid   = errors.New("redeem prepared event is invalid")
	ErrInsertRedeem           = errors.New("failed to insert redeem in statedb")
	ErrRequestedEventInvalid  = errors.New("redeem requested event is invalid")
)

type State struct {
	statedb *StateDB
	cfg     *Config

	newEthFinalizedBlockCh chan *big.Int
	newBtcFinalizedBlockCh chan *big.Int
	newMintedEventCh       chan *ethsync.MintedEvent
	newRedeemRequestedEvCh chan *ethsync.RedeemRequestedEvent
	newRedeemPreparedEvCh  chan *ethsync.RedeemPreparedEvent

	cache struct {
		lastFinalized atomic.Value // uint64
	}
}

func New(statedb *StateDB, cfg *Config) (*State, error) {
	st := &State{
		cfg:                    cfg,
		statedb:                statedb,
		newEthFinalizedBlockCh: make(chan *big.Int, 1),
		newBtcFinalizedBlockCh: make(chan *big.Int, 1),
		newMintedEventCh:       make(chan *ethsync.MintedEvent, cfg.ChannelSize),
		newRedeemRequestedEvCh: make(chan *ethsync.RedeemRequestedEvent, cfg.ChannelSize),
		newRedeemPreparedEvCh:  make(chan *ethsync.RedeemPreparedEvent, cfg.ChannelSize),
	}

	err := st.initEthFinalizedBlock()
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (st *State) Start(ctx context.Context) error {
	logger.Info("starting eth2btc state")
	defer logger.Info("stopping eth2btc state")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case blkNum := <-st.newEthFinalizedBlockCh:
			// Get the stored last finalized block number
			lastFinalized, err := st.GetEthFinalizedBlockNumber()
			if err != nil {
				logger.Errorf("failed to get last finalized block number: err=%v", err)
				return ErrGetEthFinalizedBlockNumber
			}

			// Update the last finalized block number if the new one is larger
			if lastFinalized.Cmp(blkNum) <= 0 {
				if err := st.setFinalizedBlockNumber(blkNum); err != nil {
					logger.Errorf("failed to set last finalized block number: err=%v", err)
					return ErrSetEthFinalizedBlockNumber
				}
			}
		// After receiving a redeem request event
		// 1. 	Check the existence of the redeem request tx hash
		// 2.	Skip if found
		// 3.	Insert a new redeem record in state db
		case ev := <-st.newRedeemRequestedEvCh:
			newLogger := logger.WithFields(
				"reqTx", ev.RequestTxHash.String(),
			)

			// Check if the redeem already exists
			ok, _, err := st.statedb.HasRedeem(ev.RequestTxHash)
			if err != nil {
				newLogger.Errorf("failed to check redeem existence: err=%v", err)
				return ErrCheckRedeemExistence
			}

			if ok {
				continue
			}

			// Create a new redeem and save it to the database
			redeem, err := createRedeemFromRequestedEvent(ev)
			if err != nil {
				newLogger.Errorf("failed to create redeem from requested event: err=%v, ev=%v", err, ev)
				return ErrRequestedEventInvalid
			}
			if err := st.statedb.InsertAfterRequested(redeem); err != nil {
				newLogger.Errorf("failed to insert redeem to db: err=%v", err)
				return ErrInsertRedeem
			}
			newLogger.Debug("insert redeem after requested")
		// After receiving a redeem prepared event
		// 1. 	Check the existence of the tx hash
		// 2. 	If found, check its status
		// 3.   Skip if status == prepared | completed
		// 4.   Insert the redeem if the tx hash not found or otherwise update
		// 		the existing record in db
		// NOTE that it is possible that a prepared event arrives earlier than
		// its corresponding requested event
		case ev := <-st.newRedeemPreparedEvCh:
			newLogger := logger.WithFields(
				"reqTx", ev.RequestTxHash.String(),
				"prepTx", ev.PrepareTxHash.String(),
			)

			ok, status, err := st.statedb.HasRedeem(ev.RequestTxHash)
			if err != nil {
				newLogger.Errorf("error when checking existence: err=%v", err)
				return ErrCheckRedeemExistence
			}

			var redeem *Redeem

			if ok {
				if status == RedeemStatusPrepared || status == RedeemStatusCompleted {
					continue
				}

				if status == RedeemStatusInvalid {
					newLogger.Errorf("redeem is invalid")
					return ErrRedeemInvalid
				}

				redeem, ok, err = st.statedb.GetRedeem(ev.RequestTxHash, RedeemStatusRequested)
				if err != nil || !ok {
					newLogger.Errorf("failed to get stored redeem: err=%v", err)
					return ErrGetRedeem
				}

				redeem, err = redeem.updateFromPreparedEvent(ev)
				if err != nil {
					logger.Errorf("failed to update redeem from prepared event: err=%v", err)
					return ErrPreparedEventUnmatched
				}
			} else {
				redeem, err = createRedeemFromPreparedEvent(ev)
				if err != nil {
					logger.Errorf("failed to create redeem from prepared event: err=%v, ev=%v", err, ev)
					return ErrPreparedEventInvalid
				}
			}

			if err = st.statedb.UpdateAfterPrepared(redeem); err != nil {
				return ErrUpdateRedeem
			}
			newLogger.Debug("update redeem after prepared")
		}
	}
}

func (st *State) GetEthFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, err := st.statedb.GetKeyedValue(KeyEthFinalizedBlock)
	if err != nil {
		return nil, err
	}
	st.cache.lastFinalized.Store(b.Big().Bytes())

	return b.Big(), nil
}

func (st *State) GetBtcFinalizedBlockNumber() (*big.Int, error) {
	//TODO: implement this
	return nil, nil
}

func (st *State) GetNewEthFinalizedBlockChannel() chan<- *big.Int {
	return st.newEthFinalizedBlockCh
}

func (st *State) GetNewBtcFinalizedBlockChannel() chan<- *big.Int {
	return st.newBtcFinalizedBlockCh
}

func (st *State) GetNewRedeemRequestedEventChannel() chan<- *ethsync.RedeemRequestedEvent {
	return st.newRedeemRequestedEvCh
}

func (st *State) GetNewRedeemPreparedEventChannel() chan<- *ethsync.RedeemPreparedEvent {
	return st.newRedeemPreparedEvCh
}

func (st *State) GetNewMintedEventChannel() chan<- *ethsync.MintedEvent {
	return st.newMintedEventCh
}

func (st *State) setFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.statedb.SetKeyedValue(KeyEthFinalizedBlock, common.BigInt2Bytes32(fbNum)); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(fbNum.Bytes())

	return nil
}
