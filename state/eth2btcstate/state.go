package eth2btcstate

import (
	"context"
	"database/sql"
	"math/big"
	"sync/atomic"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	MaxPendingRequestedEv = 1024
	MaxPendingPreparedEv  = 1024

	CacheSize = 1024

	KeyLastFinalizedBlock = crypto.Keccak256Hash([]byte("lastFinalizedBlock")).Bytes()
)

type State struct {
	db *StateDB

	newFinalizedCh         chan *big.Int
	newRedeemRequestedEvCh chan *ethsync.RedeemRequestedEvent
	newRedeemPreparedEvCh  chan *ethsync.RedeemPreparedEvent

	cache struct {
		lastFinalized    atomic.Value // uint64
		redeemsRequested *lru.Cache[[32]byte, *Redeem]
		redeemsPrepared  *lru.Cache[[32]byte, *Redeem]
		redeemsInvalid   *lru.Cache[[32]byte, *Redeem]
		redeemsCompleted *lru.Cache[[32]byte, *Redeem]
	}
}

var (
	stateErrors   ModifiyStateError
	stateWarnings ModifyStateWarning
)

func New(db *StateDB) (*State, error) {
	st := &State{
		db:                     db,
		newFinalizedCh:         make(chan *big.Int, 1),
		newRedeemRequestedEvCh: make(chan *ethsync.RedeemRequestedEvent, MaxPendingRequestedEv),
		newRedeemPreparedEvCh:  make(chan *ethsync.RedeemPreparedEvent, MaxPendingPreparedEv),
	}

	st.cache.redeemsRequested = lru.NewCache[[32]byte, *Redeem](CacheSize)
	st.cache.redeemsPrepared = lru.NewCache[[32]byte, *Redeem](CacheSize)
	st.cache.redeemsInvalid = lru.NewCache[[32]byte, *Redeem](CacheSize)
	st.cache.redeemsCompleted = lru.NewCache[[32]byte, *Redeem](CacheSize)

	_, err := st.db.KVGet(KeyLastFinalizedBlock)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if err == sql.ErrNoRows {
		logger.Warnf("no stored last finalized block number found, using the default value %v", common.EthStartingBlock)
		// save the default value
		db.KVSet(KeyLastFinalizedBlock, common.EthStartingBlock.Bytes())
		st.cache.lastFinalized.Store(common.EthStartingBlock.Bytes())
	} else {
		// read the stored value
		lastFinalizedBytes, err := db.KVGet(KeyLastFinalizedBlock)
		if err != nil {
			return nil, err
		}

		// compare the stored value with the default value
		lastFinalized := new(big.Int).SetBytes(lastFinalizedBytes)
		if lastFinalized.Cmp(common.EthStartingBlock) == -1 {
			return nil, stateErrors.StoredFinalizedBlockNumberLessThanStartingBlockNumber(lastFinalized)
		}
		st.cache.lastFinalized.Store(lastFinalizedBytes)
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
				return err
			}

			// Update the last finalized block number if the new one is larger
			if lastFinalized.Cmp(newFinalized) <= 0 {
				if err := st.setFinalizedBlockNumber(newFinalized); err != nil {
					return err
				}
			} else {
				logger.Warnf(stateWarnings.NewFinalizedBlockNumberLessThanStored(newFinalized, lastFinalized))
			}
		// After receiving a redeem request event
		// 1. 	Check the existence of the hash of the previous redeem requested tx.
		// 		Skip if found
		// 2.	Save a new redeem record in state db
		case ev := <-st.newRedeemRequestedEvCh:
			// Check if the redeem already exists
			ok, err := st.db.Has(ev.RedeemRequestTxHash[:])
			if err != nil {
				logger.Errorf("failed to check if redeem exists: txHash=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}

			if ok {
				logger.Warnf(stateWarnings.RedeemAlreadyExists(ev.RedeemRequestTxHash[:]))
				continue
			}

			// Create a new redeem and save it to the database
			redeem, err := createFromRequestedEvent(ev)
			if err != nil {
				logger.Errorf("failed to create redeem from requested event: err=%v, ev=%v", err, ev)
				return err
			}
			if err := st.insert(redeem); err != nil {
				logger.Errorf("failed to insert redeem to db: txHash=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}
		// After receiving a redeem prepared event
		// 1. 	Extract redeem from db using ev.RedeemRequestTxHash
		// 2. 	Return error if not existing
		// 3.   Return error if invalid
		// 4.   Skip if already prepared
		// 4.   Update redeem accordingly and save it to db
		case ev := <-st.newRedeemPreparedEvCh:
			redeem, err := st.get(ev.RedeemRequestTxHash, RedeemStatusRequested)
			if err != nil {
				logger.Errorf("failed to get redeem from db: txHash=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}

			if redeem == nil {
				return stateErrors.CannotPrepareDueToRequestedRedeemNotFound(ev.RedeemRequestTxHash[:])
			}

			if redeem.Status == RedeemStatusInvalid {
				return stateErrors.CannotPrepareDueToRequestedRedeemInvalid(ev.RedeemRequestTxHash[:])
			}

			if redeem.Status != RedeemStatusRequested {
				logger.Warnf(stateWarnings.RedeemAlreadyPreparedOrCompleted(ev.RedeemRequestTxHash[:]))
				continue
			}

			redeem, err = redeem.updateFromPreparedEvent(ev)
			if err != nil {
				logger.Errorf("failed to update redeem from prepared event: err=%v, ev=%v", err, ev)
				return err
			}
			if err = st.update(redeem); err != nil {
				return err
			}
		}
	}
}

func (st *State) GetFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, err := st.db.KVGet(KeyLastFinalizedBlock)
	if err != nil {
		return nil, err
	}
	st.cache.lastFinalized.Store(ethcommon.TrimLeftZeroes(b))

	return new(big.Int).SetBytes(b), nil
}

func (st *State) GetNewFinalizedBlockChannel() chan *big.Int {
	return st.newFinalizedCh
}

func (st *State) GetNewRedeemRequestedEventChannel() chan *ethsync.RedeemRequestedEvent {
	return st.newRedeemRequestedEvCh
}

func (st *State) GetNewRedeemPreparedEventChannel() chan *ethsync.RedeemPreparedEvent {
	return st.newRedeemPreparedEvCh
}

func (st *State) setFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.db.KVSet(KeyLastFinalizedBlock, fbNum.Bytes()); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(fbNum.Bytes())

	return nil
}

func (st *State) insert(r *Redeem) error {
	if err := st.db.InsertAfterRequested(r); err != nil {
		return err
	}

	if r.Status == RedeemStatusRequested {
		st.cache.redeemsRequested.Add(r.RequestTxHash, r.Clone())
	} else if r.Status == RedeemStatusInvalid {
		st.cache.redeemsInvalid.Add(r.RequestTxHash, r.Clone())
	}

	return nil
}

func (st *State) update(r *Redeem) error {
	if err := st.db.UpdateAfterPrepared(r); err != nil {
		return err
	}

	st.cache.redeemsPrepared.Add(r.RequestTxHash, r.Clone())
	st.cache.redeemsRequested.Remove(r.RequestTxHash)

	return nil
}

func (st *State) get(ethTxHash [32]byte, status RedeemStatus) (*Redeem, error) {
	if status == RedeemStatusRequested {
		if v, ok := st.cache.redeemsRequested.Get(ethTxHash); ok {
			return v, nil
		}
	}

	if status == RedeemStatusPrepared {
		if v, ok := st.cache.redeemsPrepared.Get(ethTxHash); ok {
			return v, nil
		}
	}

	if status == RedeemStatusInvalid {
		if v, ok := st.cache.redeemsInvalid.Get(ethTxHash); ok {
			return v, nil
		}
	}

	if status == RedeemStatusCompleted {
		if v, ok := st.cache.redeemsCompleted.Get(ethTxHash); ok {
			return v, nil
		}
	}

	r, err := st.db.Get(ethTxHash[:], status)
	if err != nil {
		return nil, err
	}

	if r == nil {
		return nil, nil
	}

	if r.Status == RedeemStatusRequested {
		st.cache.redeemsRequested.Add(r.RequestTxHash, r.Clone())
	}

	if r.Status == RedeemStatusPrepared {
		st.cache.redeemsPrepared.Add(r.RequestTxHash, r.Clone())
	}

	if r.Status == RedeemStatusInvalid {
		st.cache.redeemsInvalid.Add(r.RequestTxHash, r.Clone())
	}

	if r.Status == RedeemStatusCompleted {
		st.cache.redeemsCompleted.Add(r.RequestTxHash, r.Clone())
	}

	return r, nil
}
