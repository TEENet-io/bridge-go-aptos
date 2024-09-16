package eth2btcstate

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
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
	ctx    context.Context
	cancel context.CancelFunc

	db Database

	newFinalizedCh         chan *big.Int
	newRedeemRequestedEvCh chan *ethsync.RedeemRequestedEvent
	newRedeemPreparedEvCh  chan *ethsync.RedeemPreparedEvent

	cache struct {
		lastFinalized atomic.Value // uint64
		redeems       *lru.Cache[[32]byte, *Redeem]
	}
}

func New(db Database) (*State, error) {
	st := &State{
		db:                     db,
		newFinalizedCh:         make(chan *big.Int, 1),
		newRedeemRequestedEvCh: make(chan *ethsync.RedeemRequestedEvent, MaxPendingRequestedEv),
		newRedeemPreparedEvCh:  make(chan *ethsync.RedeemPreparedEvent, MaxPendingPreparedEv),
	}

	st.cache.redeems = lru.NewCache[[32]byte, *Redeem](CacheSize)

	ok, err := st.db.Has(KeyLastFinalizedBlock)
	if err != nil {
		return nil, err
	}

	if !ok {
		// save the default value
		db.Put(KeyLastFinalizedBlock, common.EthStartingBlock.Bytes())
		st.cache.lastFinalized.Store(common.EthStartingBlock.Bytes())
	} else {
		// read the stored value
		lastFinalizedBytes, err := db.Get(KeyLastFinalizedBlock)
		if err != nil {
			return nil, err
		}

		// compare the stored value with the default value
		lastFinalized := new(big.Int).SetBytes(lastFinalizedBytes)
		if lastFinalized.Cmp(common.EthStartingBlock) == -1 {
			logger.Errorf("stored value %s less than the starting block number %s",
				lastFinalized.Text(10), common.EthStartingBlock.Text(10))
			st.cache.lastFinalized.Store(common.EthStartingBlock.Bytes())
		}
		st.cache.lastFinalized.Store(lastFinalizedBytes)
	}

	return st, nil
}

func (st *State) Start(ctx context.Context) error {
	st.ctx, st.cancel = context.WithCancel(context.Background())

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
				logger.Warnf("new finalized block number %s less than the stored one %s",
					newFinalized.Text(10), lastFinalized.Text(10))
			}
		// After receiving a redeem request event
		// 1. 	Check the existence of the hash of the previous redeem requested tx.
		// 		Skip if found
		// 2.	Save a new redeem record in state db
		case ev := <-st.newRedeemRequestedEvCh:
			// Check if the redeem already exists
			ok, err := st.has(ev.RedeemRequestTxHash)
			if err != nil {
				return err
			}

			if ok {
				logger.Warnf("redeem %s already exists", common.Shorten(common.Bytes32ToHexStr(ev.RedeemRequestTxHash)))
				continue
			}

			// Create a new redeem and save it to the database
			redeem := (&Redeem{}).SetFromRequestedEvent(ev)
			if err := st.put(redeem); err != nil {
				return err
			}
		// After receiving a redeem prepared event
		// 1. 	Extract redeem from db using ev.RedeemRequestTxHash
		// 2. 	Return error if not existing
		// 3.   Return error if invalid
		// 4.   Skip if already prepared
		// 4.   Update redeem accordingly and save it to db
		case ev := <-st.newRedeemPreparedEvCh:
			redeem, err := st.get(ev.RedeemRequestTxHash)
			if err != nil {
				return err
			}

			if redeem == nil {
				return errors.New(ErrorRedeemNotFound)
			}

			if redeem.Status == RedeemStatusInvalid {
				return errors.New(ErrorRedeemInvalid)
			}

			if redeem.Status != RedeemStatusRequested {
				continue
			}

			err = st.put(redeem.SetFromPreparedEvent(ev))
			if err != nil {
				return err
			}
		}
	}
}

func (st *State) Stop() {
	st.cancel()
}

func (st *State) GetFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, err := st.db.Get(KeyLastFinalizedBlock)
	if err != nil {
		return nil, err
	}
	st.cache.lastFinalized.Store(b)

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

func (st *State) setFinalizedBlockNumber(blk *big.Int) error {
	if err := st.db.Put(KeyLastFinalizedBlock, blk.Bytes()); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(blk.Bytes())

	return nil
}

func (st *State) put(r *Redeem) error {
	b, err := r.MarshalJSON()
	if err != nil {
		return err
	}

	if err := st.db.Put(r.RequestTxHash[:], b); err != nil {
		return err
	}

	st.cache.redeems.Add(r.RequestTxHash, r.Clone())

	return nil
}

func (st *State) get(ethTxHash [32]byte) (*Redeem, error) {
	ok, err := st.has(ethTxHash)
	if err != nil {
		return nil, err
	}

	if ok {
		if r, ok := st.cache.redeems.Get(ethTxHash); ok {
			return r.Clone(), nil
		}
	}

	return nil, nil
}

func (st *State) has(ethTxHash [32]byte) (bool, error) {
	if st.cache.redeems.Contains(ethTxHash) {
		return true, nil
	}

	ok, err := st.db.Has(ethTxHash[:])
	if err != nil {
		return false, err
	}

	if ok {
		b, err := st.db.Get(ethTxHash[:])
		if err != nil {
			return false, err
		}
		redeem := &Redeem{}
		err = redeem.UnmarshalJSON(b)
		if err != nil {
			return false, err
		}

		st.cache.redeems.Add(ethTxHash, redeem)

		return true, nil
	}

	return false, nil
}
