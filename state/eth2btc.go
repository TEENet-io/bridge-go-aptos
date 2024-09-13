package state

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

var (
	MaxPendingRequestedEv = 1024
	MaxPendingPreparedEv  = 1024

	CacheSize = 1024

	KeyLastFinalizedBlock = crypto.Keccak256Hash([]byte("lastFinalizedBlock")).Bytes()
)

type E2BState struct {
	db ethdb.Database

	lastFinalizedCh chan *big.Int
	requestedEvCh   chan *etherman.RedeemRequestedEvent
	preparedEvCh    chan *etherman.RedeemPreparedEvent

	cache struct {
		lastFinalized atomic.Value // uint4
		redeems       *lru.Cache[string, *Redeem]
	}
}

func NewEth2BtcState(db ethdb.Database) (*E2BState, error) {
	st := &E2BState{
		db:              db,
		lastFinalizedCh: make(chan *big.Int, 1),
		requestedEvCh:   make(chan *etherman.RedeemRequestedEvent, MaxPendingRequestedEv),
		preparedEvCh:    make(chan *etherman.RedeemPreparedEvent, MaxPendingPreparedEv),
	}

	st.cache.redeems = lru.NewCache[string, *Redeem](CacheSize)

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

func (st *E2BState) Start(ctx context.Context) error {
	logger.Info("starting eth2btc state")
	defer logger.Info("stopping eth2btc state")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case newFinalized := <-st.lastFinalizedCh:
			// Get the stored last finalized block number
			lastFinalized, err := st.GetFinalizedBlockNumber()
			if err != nil {
				return err
			}

			// Update the last finalized block number if the new one is larger
			if lastFinalized.Cmp(newFinalized) <= 0 {
				if err := st.setFinalizedBlockNumber(newFinalized); err != nil {
					logger.Infof("set new finalized block number %s", newFinalized.Text(10))
					return err
				}
			} else {
				logger.Warnf("new finalized block number %s less than the stored one %s",
					newFinalized.Text(10), lastFinalized.Text(10))
			}
		case ev := <-st.requestedEvCh:
			// Check if the redeem already exists
			ok, err := st.hasRedeem(ev.TxHash)
			if err != nil {
				return err
			}

			// If the redeem already exists, log a warning and continue
			if ok {
				logger.Warnf("redeem %s already exists", common.Shorten(common.Bytes32ToHexStr(ev.TxHash)))
				continue
			}

			// Create a new redeem and save it to the database
			redeem := (&Redeem{}).SetFromRequestedEvent(ev)
			if err := st.setRedeem(redeem); err != nil {
				return err
			}
		case ev := <-st.preparedEvCh:
			ok, err := st.hasRedeem(ev.EthTxHash)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New(RedeemNotFound)
			}

			redeem, err := st.getRedeem(ev.EthTxHash)
			if err != nil {
				return err
			}

			if redeem.HasPrepared() {
				logger.Warnf("redeem %s already prepared", common.Shorten(common.Bytes32ToHexStr(ev.EthTxHash)))
				continue
			}

			err = st.setRedeem(redeem.SetFromPreparedEvent(ev))
			if err != nil {
				return err
			}
		}
	}
}

func (st *E2BState) GetFinalizedBlockNumber() (*big.Int, error) {
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

func (st *E2BState) GetLastEthFinalizedBlockNumberChannel() chan *big.Int {
	return st.lastFinalizedCh
}

func (st *E2BState) GetRedeemRequestedEventChannel() chan *etherman.RedeemRequestedEvent {
	return st.requestedEvCh
}

func (st *E2BState) GetRedeemPreparedEventChannel() chan *etherman.RedeemPreparedEvent {
	return st.preparedEvCh
}

func (st *E2BState) setFinalizedBlockNumber(blk *big.Int) error {
	if err := st.db.Put(KeyLastFinalizedBlock, blk.Bytes()); err != nil {
		return err
	}
	st.cache.lastFinalized.Store(blk.Bytes())

	return nil
}

func (st *E2BState) setRedeem(r *Redeem) error {
	b, err := r.MarshalJSON()
	if err != nil {
		return err
	}

	if err := st.db.Put(r.RequestTxHash[:], b); err != nil {
		return err
	}

	fmt.Println(st.db.Has(r.RequestTxHash[:]))

	st.cache.redeems.Add(common.Bytes32ToHexStr(r.RequestTxHash), r.Clone())

	return nil
}

func (st *E2BState) getRedeem(ethTxHash [32]byte) (*Redeem, error) {
	ok, err := st.hasRedeem(ethTxHash)
	if err != nil {
		return nil, err
	}

	if ok {
		if r, ok := st.cache.redeems.Get(common.Bytes32ToHexStr(ethTxHash)); ok {
			return r.Clone(), nil
		}
	}

	return nil, nil
}

func (st *E2BState) hasRedeem(ethTxHash [32]byte) (bool, error) {
	id := common.Bytes32ToHexStr(ethTxHash)

	if st.cache.redeems.Contains(id) {
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

		st.cache.redeems.Add(id, redeem)

		return true, nil
	}

	return false, nil
}
