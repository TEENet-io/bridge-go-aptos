package eth2btcstate

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"sync/atomic"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	KeyLastFinalizedBlock = crypto.Keccak256Hash([]byte("lastFinalizedBlock"))

	ErrRedeemInvalid                     = errors.New("redeem is invalid")
	ErrStoredFinalizedBlockNumberInvalid = errors.New("stored finalized block number is invalid")
	ErrGetRedeem                         = errors.New("failed to get redeem from statedb")
	ErrCheckRedeemExistence              = errors.New("failed to check redeem existence")
	ErrPreparedEventUnmatched            = errors.New("redeem prepared event is unmatched with stored requested redeem")
	ErrUpdateRedeem                      = errors.New("failed to update redeem in statedb")
	ErrPreparedEventInvalid              = errors.New("redeem prepared event is invalid")
	ErrInsertRedeem                      = errors.New("failed to insert redeem in statedb")
	ErrRequestedEventInvalid             = errors.New("redeem requested event is invalid")
	ErrSetFinalizedBlockNumber           = errors.New("failed to set finalized block number in statedb")
	ErrGetFinalizedBlockNumber           = errors.New("failed to get finalized block number from statedb")
)

type State struct {
	db  *StateDB
	cfg *Config

	newFinalizedCh         chan *big.Int
	newRedeemRequestedEvCh chan *ethsync.RedeemRequestedEvent
	newRedeemPreparedEvCh  chan *ethsync.RedeemPreparedEvent

	cache struct {
		lastFinalized atomic.Value // uint64
	}
}

func New(db *StateDB, cfg *Config) (*State, error) {
	st := &State{
		db:                     db,
		cfg:                    cfg,
		newFinalizedCh:         make(chan *big.Int, 1),
		newRedeemRequestedEvCh: make(chan *ethsync.RedeemRequestedEvent, cfg.ChannelSize),
		newRedeemPreparedEvCh:  make(chan *ethsync.RedeemPreparedEvent, cfg.ChannelSize),
	}

	storedBytes32, err := st.db.GetKeyedValue(KeyLastFinalizedBlock)
	if err != nil && err != sql.ErrNoRows {
		return nil, ErrGetFinalizedBlockNumber
	}

	if err == sql.ErrNoRows {
		logger.Warnf("no stored last finalized block number found, using the default value %v", common.EthStartingBlock)
		// save the default value
		err := db.setKeyedValue(KeyLastFinalizedBlock, common.BigInt2Bytes32(common.EthStartingBlock))
		if err != nil {
			return nil, ErrSetFinalizedBlockNumber
		}
		st.cache.lastFinalized.Store(common.EthStartingBlock.Bytes())
	} else {
		stored := new(big.Int).SetBytes(storedBytes32[:])

		// stored value must not be less than the starting block number
		if stored.Cmp(common.EthStartingBlock) == -1 {
			logger.Errorf("stored last finalized block number is invalid: %v", stored)
			return nil, ErrStoredFinalizedBlockNumberInvalid
		}
		st.cache.lastFinalized.Store(stored)
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
		case newFinalized := <-st.newFinalizedCh:
			// Get the stored last finalized block number
			lastFinalized, err := st.GetFinalizedBlockNumber()
			if err != nil {
				logger.Errorf("failed to get last finalized block number: err=%v", err)
				return ErrGetFinalizedBlockNumber
			}

			// Update the last finalized block number if the new one is larger
			if lastFinalized.Cmp(newFinalized) <= 0 {
				if err := st.setFinalizedBlockNumber(newFinalized); err != nil {
					logger.Errorf("failed to set last finalized block number: err=%v", err)
					return ErrSetFinalizedBlockNumber
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
			ok, _, err := st.db.Has(ev.RequestTxHash)
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
			if err := st.db.insertAfterRequested(redeem); err != nil {
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

			ok, status, err := st.db.Has(ev.RequestTxHash)
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

				redeem, ok, err = st.db.Get(ev.RequestTxHash, RedeemStatusRequested)
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

			if err = st.db.updateAfterPrepared(redeem); err != nil {
				return ErrUpdateRedeem
			}
			newLogger.Debug("update redeem after prepared")
		}
	}
}

func (st *State) GetFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, err := st.db.GetKeyedValue(KeyLastFinalizedBlock)
	if err != nil {
		return nil, err
	}
	st.cache.lastFinalized.Store(b.Big().Bytes())

	return b.Big(), nil
}

func (st *State) GetNewFinalizedBlockChannel() chan<- *big.Int {
	return st.newFinalizedCh
}

func (st *State) GetNewRedeemRequestedEventChannel() chan<- *ethsync.RedeemRequestedEvent {
	return st.newRedeemRequestedEvCh
}

func (st *State) GetNewRedeemPreparedEventChannel() chan<- *ethsync.RedeemPreparedEvent {
	return st.newRedeemPreparedEvCh
}

func (st *State) setFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.db.setKeyedValue(KeyLastFinalizedBlock, common.BigInt2Bytes32(fbNum)); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(fbNum.Bytes())

	return nil
}
