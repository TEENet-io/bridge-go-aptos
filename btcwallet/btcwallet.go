package btcwallet

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/ethtxmanager"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrNotFound                = errors.New("no spendables found")
	ErrDBOpGetRequestsByStatus = errors.New("failed to get requests by status")
)

type BtcWallet struct {
	cfg   *Config
	db    *BtcWalletDB
	mgrdb *ethtxmanager.EthTxManagerDB

	newSpendableCh chan interface{}

	// mutex for reading and writing spendable outpoints in db
	mu sync.Mutex
}

func NewBtcWallet(cfg *Config, db *BtcWalletDB, mgrdb *ethtxmanager.EthTxManagerDB) *BtcWallet {
	return &BtcWallet{
		cfg:            cfg,
		db:             db,
		mgrdb:          mgrdb,
		newSpendableCh: make(chan interface{}, 1),
	}
}

func (w *BtcWallet) Close() {
	w.db.Close()
}

func (w *BtcWallet) Start(ctx context.Context) error {
	logger.Info("btc wallet started")
	defer logger.Info("btc wallet stopped")

	errCh := make(chan error, 1)
	defer close(errCh)

	ticker := time.NewTicker(w.cfg.FrequencyToCheckRequests)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			switch err {
			case ErrDBOpGetRequestsByStatus:
			default:
				logger.Fatalf("unexpected error: %v", err)
			}
			return err
		case <-w.newSpendableCh:
			// TODO: handle new spendable sent by btc sychronizer
		case <-ticker.C:
			// TODO: check requests and lock corresponding spendables if they fail
			reqs, err := w.db.GetRequestsByStatus(Locked)
			if err != nil {
				logger.Errorf("failed to get requests: %v", err)
				errCh <- ErrDBOpGetRequestsByStatus
			}

			if len(reqs) == 0 {
				continue
			}
		}
	}
}

func (w *BtcWallet) Request(id ethcommon.Hash, amount *big.Int, ch chan<- []state.Outpoint) error {
	// Set the lock to prevent multiple go routines from simulataneously
	// requesting spenable outpoints
	w.mu.Lock()
	defer w.mu.Unlock()

	newLogger := logger.WithFields(
		"id", id,
		"amount", amount,
	)

	spendables, ok, err := w.db.RequestSpendablesByAmount(amount)
	if err != nil {
		newLogger.Errorf("failed to get spendables: %v", err)
		return err
	}

	// not found
	if !ok {
		newLogger.Errorf("no spendables found")
		return ErrNotFound
	}

	// prepare outpoint data
	sum := big.NewInt(0)
	outpoints := []state.Outpoint{}
	for _, spendable := range spendables {
		sum.Add(sum, spendable.Amount)
		outpoints = append(outpoints, state.Outpoint{
			TxId: spendable.BtcTxId,
			Idx:  spendable.Idx,
		})
	}

	// send back to the requester via channel
	newLogger.Info("sending outpoints to requester: sum=%v, requested=%v", sum, amount)
	ch <- outpoints

	// lock the spenables
	for _, spendable := range spendables {
		err := w.db.SetLockOnSpendable(spendable.BtcTxId, true)
		if err != nil {
			newLogger.Errorf("failed to set lock on spendable: %v", err)
			return err
		}
	}

	// save the request with status = locked info to db
	if err := w.db.InsertRequest(&Request{
		Id:        id,
		Outpoints: outpoints,
		Status:    Locked,
	}); err != nil {
		newLogger.Errorf("failed to insert request: %v", err)
		return err
	}

	return nil
}
