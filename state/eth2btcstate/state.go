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
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	KeyLastFinalizedBlock = crypto.Keccak256Hash([]byte("lastFinalizedBlock")).Bytes()
)

type State struct {
	db  *StateDB
	cfg *Config

	newFinalizedCh         chan *big.Int
	newRedeemRequestedEvCh chan *ethsync.RedeemRequestedEvent
	newRedeemPreparedEvCh  chan *ethsync.RedeemPreparedEvent

	cache struct {
		lastFinalized atomic.Value // uint64
		redeems       *redeemCache
	}
}

var (
	stateErrors StateError
)

func New(db *StateDB, cfg *Config) (*State, error) {
	st := &State{
		db:                     db,
		cfg:                    cfg,
		newFinalizedCh:         make(chan *big.Int, 1),
		newRedeemRequestedEvCh: make(chan *ethsync.RedeemRequestedEvent, cfg.ChannelSize),
		newRedeemPreparedEvCh:  make(chan *ethsync.RedeemPreparedEvent, cfg.ChannelSize),
	}
	st.cache.redeems = newRedeemCache(cfg.CacheSize)

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

func (st *State) Close() {
	if st.db != nil {
		st.db.close()
	}
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
			}
		// After receiving a redeem request event
		// 1. 	Check the existence of the redeem request tx hash
		// 2.	Skip if found
		// 3.	Insert a new redeem record in state db
		case ev := <-st.newRedeemRequestedEvCh:
			// Check if the redeem already exists
			ok, _, err := st.db.has(ev.RedeemRequestTxHash[:])
			if err != nil {
				logger.Errorf("failed to check if redeem exists: tx=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}

			if ok {
				continue
			}

			// Create a new redeem and save it to the database
			redeem, err := createRedeemFromRequestedEvent(ev)
			if err != nil {
				logger.Errorf("failed to create redeem from requested event: err=%v, ev=%v", err, ev)
				return err
			}
			if err := st.insert(redeem); err != nil {
				logger.Errorf("failed to insert redeem to db: tx=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}
		// After receiving a redeem prepared event
		// 1. 	Check the existence of the tx hash
		// 2. 	If found, check its status
		// 3.   Skip if status == prepared | completed
		// 4.   Insert the redeem if the tx hash not found or otherwise update
		// 		the existing record in db
		// NOTE that it is possible that a prepared event arrives earlier than
		// its corresponding requested event
		case ev := <-st.newRedeemPreparedEvCh:
			ok, status, err := st.db.has(ev.RedeemRequestTxHash[:])
			if err != nil {
				logger.Errorf("failed to check if redeem exists: tx=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				return err
			}

			var redeem *Redeem

			if ok {
				if status == RedeemStatusPrepared || status == RedeemStatusCompleted {
					continue
				}

				if status == RedeemStatusInvalid {
					logger.Errorf("cannot prepare since redeem request is invalid: tx=0x%x, status=%s", ev.RedeemRequestTxHash[:], status)
					return stateErrors.CannotPrepareDueToRedeemRequestInvalid(ev.RedeemRequestTxHash[:])
				}

				redeem, _, err = st.Get(ev.RedeemRequestTxHash, RedeemStatusRequested)
				if err != nil {
					logger.Errorf("failed to get stored redeem: tx=0x%x, err=%v", ev.RedeemRequestTxHash[:], err)
				}

				redeem, err = redeem.updateFromPreparedEvent(ev)
				if err != nil {
					logger.Errorf("failed to update redeem from prepared event: err=%v, ev=%v", err, ev)
					return err
				}
			} else {
				redeem, err = createRedeemFromPreparedEvent(ev)
				if err != nil {
					logger.Errorf("failed to create redeem from prepared event: err=%v, ev=%v", err, ev)
					return err
				}
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

func (st *State) GetNewFinalizedBlockChannel() chan<- *big.Int {
	return st.newFinalizedCh
}

func (st *State) GetNewRedeemRequestedEventChannel() chan<- *ethsync.RedeemRequestedEvent {
	return st.newRedeemRequestedEvCh
}

func (st *State) GetNewRedeemPreparedEventChannel() chan<- *ethsync.RedeemPreparedEvent {
	return st.newRedeemPreparedEvCh
}

func (st *State) Get(ethTxHash [32]byte, status RedeemStatus) (*Redeem, bool, error) {
	if v, ok := st.cache.redeems.get(ethTxHash, status); ok {
		return v, true, nil
	}

	r, ok, err := st.db.get(ethTxHash[:], status)
	if err != nil {
		return nil, false, err
	}

	if ok {
		st.cache.redeems.add(r)
	}

	return r, true, nil
}

func (st *State) GetByStatus(status RedeemStatus) ([]*Redeem, error) {
	redeems, err := st.db.getByStatus(status)
	if err != nil {
		return nil, err
	}

	for _, r := range redeems {
		st.cache.redeems.add(r)
	}

	return redeems, nil
}

func (st *State) setFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.db.KVSet(KeyLastFinalizedBlock, fbNum.Bytes()); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(fbNum.Bytes())

	return nil
}

func (st *State) insert(r *Redeem) error {
	if err := st.db.insertAfterRequested(r); err != nil {
		return err
	}

	st.cache.redeems.add(r)

	return nil
}

func (st *State) update(r *Redeem) error {
	if err := st.db.updateAfterPrepared(r); err != nil {
		return err
	}

	st.cache.redeems.remove(r.RequestTxHash, RedeemStatusRequested)
	st.cache.redeems.add(r)

	return nil
}
