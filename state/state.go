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
	KeyEthChainId        = crypto.Keccak256Hash([]byte("KeyEthChainId"))

	ErrSetBtcFinalizedBlockNumber           = errors.New("failed to set btc finalized block number in statedb")
	ErrGetBtcFinalizedBlockNumber           = errors.New("failed to get btc finalized block number from statedb")
	ErrSetEthFinalizedBlockNumber           = errors.New("failed to set eth finalized block number in statedb")
	ErrGetEthFinalizedBlockNumber           = errors.New("failed to get eth finalized block number from statedb")
	ErrStoredEthFinalizedBlockNumberInvalid = errors.New("stored eth finalized block number is invalid")
	ErrKeyValueNotFound                     = errors.New("key value not found")
	ErrGetEthChainId                        = errors.New("failed to get eth chain id from statedb")
	ErrSetEthChainId                        = errors.New("failed to set eth chain id in statedb")
	ErrEthChainIdUnmatchedStored            = errors.New("current chain id does not match the stored")

	ErrUpdateInvalidRedeem    = errors.New("redeem is invalid and cannot be updated")
	ErrPreparedEventUnmatched = errors.New("redeem prepared event is unmatched with stored requested redeem")
	ErrPreparedEventInvalid   = errors.New("redeem prepared event is invalid")
	ErrRequestedEventInvalid  = errors.New("redeem requested event is invalid")

	ErrDBOpUpdateRedeem = errors.New("failed to update redeem in statedb")
	ErrDBOpGetRedeem    = errors.New("failed to get redeem from statedb")
	ErrDBOpHasRedeem    = errors.New("failed to check redeem existence")
	ErrDBOpInsertRedeem = errors.New("failed to insert redeem in statedb")
	ErrDBOpUpdateMint   = errors.New("failed to update mint in statedb")
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
		lastEthFinalized atomic.Value // uint64
		lastBtcFinalized atomic.Value // uint64
		ethChainId       atomic.Value // uint64
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

	if err := st.initEthFinalizedBlock(); err != nil {
		return nil, err
	}

	if err := st.initEthChainID(); err != nil {
		return nil, err
	}

	// TODO: init btc finalized block number

	return st, nil
}

func (st *State) Start(ctx context.Context) error {
	logger.Info("starting state")
	defer logger.Info("stopping state")

	// TODO: error handling

	errCh := make(chan error, 1)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			switch err {
			case ErrGetEthFinalizedBlockNumber:
			case ErrSetEthFinalizedBlockNumber:
			case ErrDBOpUpdateMint:
			case ErrDBOpHasRedeem:
			case ErrDBOpInsertRedeem:
			case ErrRequestedEventInvalid:
			case ErrDBOpGetRedeem:
			case ErrDBOpUpdateRedeem:
			case ErrPreparedEventInvalid:
			case ErrPreparedEventUnmatched:
			case ErrUpdateInvalidRedeem:
			default:
				logger.Fatal(err)
			}
			return err
		case blkNum := <-st.newEthFinalizedBlockCh:
			newLogger := logger.WithFields("newFinalized", blkNum.String())

			handleNewBlockNumber := func() error {
				// Get the stored last finalized block number
				lastFinalized, err := st.GetEthFinalizedBlockNumber()
				if err != nil {
					newLogger.Errorf("failed to get last finalized block number: err=%v", err)
					return ErrGetEthFinalizedBlockNumber
				}

				// Update the last finalized block number if the new one is larger
				if lastFinalized.Cmp(blkNum) <= 0 {
					if err := st.setEthFinalizedBlockNumber(blkNum); err != nil {
						newLogger.Errorf("failed to set last finalized block number: err=%v", err)
						return ErrSetEthFinalizedBlockNumber
					}
				}
				return nil
			}

			if err := handleNewBlockNumber(); err != nil {
				errCh <- err
			}
		// After receiving a new minted event, udpate statedb
		case ev := <-st.newMintedEventCh:
			newLogger := logger.WithFields(
				"mintTx", ev.MintTxHash.String(),
				"btcTxId", ev.BtcTxId.String(),
			)

			handleEvent := func() error {
				mint := createMintFromMintedEvent(ev)

				err := st.statedb.UpdateMint(mint)
				if err != nil {
					newLogger.Errorf("failed to update mint: err=%v", err)
					return ErrDBOpUpdateMint
				}
				newLogger.Debug("update mint")
				return nil
			}

			if err := handleEvent(); err != nil {
				errCh <- err
			}
		// After receiving a redeem request event
		// 1. 	Check the existence of the redeem request tx hash
		// 2.	Skip if found
		// 3.	Insert a new redeem record in state db
		case ev := <-st.newRedeemRequestedEvCh:
			newLogger := logger.WithFields(
				"reqTx", ev.RequestTxHash.String(),
			)

			handleEvent := func() error {
				// Check if the redeem already exists
				ok, _, err := st.statedb.HasRedeem(ev.RequestTxHash)
				if err != nil {
					newLogger.Errorf("failed to check redeem existence: err=%v", err)
					return ErrDBOpHasRedeem
				}

				if ok {
					return nil
				}

				// Create a new redeem and save it to the database
				redeem, err := createRedeemFromRequestedEvent(ev)
				if err != nil {
					newLogger.Errorf("failed to create redeem from requested event: err=%v, ev=%v", err, ev)
					return ErrRequestedEventInvalid
				}
				if err := st.statedb.InsertAfterRequested(redeem); err != nil {
					newLogger.Errorf("failed to insert redeem to db: err=%v", err)
					return ErrDBOpInsertRedeem
				}
				newLogger.Debug("insert redeem after requested")

				return nil
			}

			if err := handleEvent(); err != nil {
				errCh <- err
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
			newLogger := logger.WithFields(
				"reqTx", ev.RequestTxHash.String(),
				"prepTx", ev.PrepareTxHash.String(),
			)

			handleEvent := func() error {
				ok, status, err := st.statedb.HasRedeem(ev.RequestTxHash)
				if err != nil {
					newLogger.Errorf("error when checking existence: err=%v", err)
					return ErrDBOpHasRedeem
				}

				var redeem *Redeem

				if ok {
					// Even though the "RedeemPrepare" is newly mined on ETH chain,
					// correspoinding state DB record should be still "requested" status.
					// If found "prepared" or "completed" status, skip the event.
					if status == RedeemStatusPrepared || status == RedeemStatusCompleted {
						return nil
					}

					if status == RedeemStatusInvalid {
						newLogger.Errorf("redeem is invalid and cannot be updated")
						return ErrUpdateInvalidRedeem
					}

					redeem, ok, err = st.statedb.GetRedeem(ev.RequestTxHash)
					if err != nil || !ok {
						newLogger.Errorf("failed to get stored redeem: err=%v", err)
						return ErrDBOpGetRedeem
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
					return ErrDBOpUpdateRedeem
				}
				newLogger.Debug("update redeem after prepared")

				return nil
			}

			if err := handleEvent(); err != nil {
				errCh <- err
			}
		}
	}
}

// Fetch latest finalized eth block number from statedb
func (st *State) GetEthFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastEthFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, ok, err := st.statedb.GetKeyedValue(KeyEthFinalizedBlock)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrKeyValueNotFound
	}
	st.cache.lastEthFinalized.Store(b.Big().Bytes())

	return b.Big(), nil
}

// Set finalized ETH block number into state (update underlying db).
func (st *State) setEthFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.statedb.SetKeyedValue(KeyEthFinalizedBlock, common.BigInt2Bytes32(fbNum)); err != nil {
		return err
	}
	st.cache.lastEthFinalized.Store(fbNum.Bytes())

	return nil
}

// Fetch latest btc finalized block number from statedb
func (st *State) GetBtcFinalizedBlockNumber() (*big.Int, error) {
	if v := st.cache.lastBtcFinalized.Load(); v != nil {
		return new(big.Int).SetBytes(v.([]byte)), nil
	}

	b, ok, err := st.statedb.GetKeyedValue(KeyBtcFinalizedBlock)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrKeyValueNotFound
	}
	st.cache.lastBtcFinalized.Store(b.Big().Bytes())

	return b.Big(), nil
}

// Set finalized BTC block number into state (update underlying db).
func (st *State) SetBtcFinalizedBlockNumber(fbNum *big.Int) error {
	if err := st.statedb.SetKeyedValue(KeyBtcFinalizedBlock, common.BigInt2Bytes32(fbNum)); err != nil {
		return err
	}
	st.cache.lastBtcFinalized.Store(fbNum.Bytes())

	return nil
}

// Return a channel
func (st *State) GetNewEthFinalizedBlockChannel() chan<- *big.Int {
	return st.newEthFinalizedBlockCh
}

// Return a channel
func (st *State) GetNewBtcFinalizedBlockChannel() chan<- *big.Int {
	return st.newBtcFinalizedBlockCh
}

// Return a channel
func (st *State) GetNewRedeemRequestedEventChannel() chan<- *ethsync.RedeemRequestedEvent {
	return st.newRedeemRequestedEvCh
}

// Return a channel
func (st *State) GetNewRedeemPreparedEventChannel() chan<- *ethsync.RedeemPreparedEvent {
	return st.newRedeemPreparedEvCh
}

// Return a channel
func (st *State) GetNewMintedEventChannel() chan<- *ethsync.MintedEvent {
	return st.newMintedEventCh
}

// Insert a new BTC2EVM mint record into state db.
func (st *State) SetNewBTC2EVMMint(m *Mint) error {
	err := st.statedb.UpdateMint(m)
	return err
}

// GetPreparedRedeems fetches all redeems with status "prepared" from the statedb.
func (st *State) GetPreparedRedeems() ([]*Redeem, error) {
	redeems, err := st.statedb.GetRedeemsByStatus(RedeemStatusPrepared)
	if err != nil {
		return nil, err
	}
	return redeems, nil
}

// Update existing redeem record in the state db.
// Set the status to from "prepared" to "completed"
func (st *State) SetRedeemCompleted(r *Redeem) error {
	return st.statedb.UpdateAfterRedeemed(r)
}
