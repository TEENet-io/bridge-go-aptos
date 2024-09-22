package ethtxmanager

import (
	"context"
	"errors"
	"sync"
	"time"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	// error when the eth tx manager times out on waiting for a signature request
	ErrTimedOutOnWaitingForSignature = errors.New("timed out on waiting for signature")

	ErrInvalidSchnorrSignature = errors.New("invalid schnorr signature")

	ErrEmptyOutpointsReturned = errors.New("empty outpoints returned")
)

type EthTxManager struct {
	ctx      context.Context
	cfg      *Config
	etherman *etherman.Etherman
	statedb  *eth2btcstate.StateDB
	mgrdb    *EthTxManagerDB

	schnorrWallet SchnorrThresholdWallet
	btcWallet     BtcWallet

	// public key of the schnorr threshold signature
	pubKey [32]byte

	redeemWaitingSignature sync.Map
}

func New(
	ctx context.Context,
	cfg *Config,
	etherman *etherman.Etherman,
	statedb *eth2btcstate.StateDB,
	mgrdb *EthTxManagerDB,
	schnorrWallet SchnorrThresholdWallet,
	btcWallet BtcWallet,
) (*EthTxManager, error) {
	// Get the public key of the schnorr threshold signature
	pk, err := etherman.GetPublicKey()
	if err != nil {
		logger.Errorf("failed to get public key: err=%v", err)
		return nil, err
	}
	pubKey := common.BigInt2Bytes32(pk)

	return &EthTxManager{
		ctx:           ctx,
		etherman:      etherman,
		statedb:       statedb,
		cfg:           cfg,
		mgrdb:         mgrdb,
		pubKey:        pubKey,
		schnorrWallet: schnorrWallet,
		btcWallet:     btcWallet,
	}, nil

}

func (txmgr *EthTxManager) Start(ctx context.Context) error {
	logger.Info("starting eth tx manager")
	defer logger.Info("stopping eth tx manager")

	for {
		select {
		case <-txmgr.ctx.Done():
			return txmgr.ctx.Err()
		case <-time.After(txmgr.cfg.FrequencyToGetUnpreparedRedeem):
			redeems, err := txmgr.statedb.GetByStatus(eth2btcstate.RedeemStatusRequested)
			if err != nil {
				logger.Error("DB: failed to get redeems by status")
				return err
			}

			for _, redeem := range redeems {
				logger1 := logger.WithFields("requestTxHash", redeem.RequestTxHash.String())

				// Check whether there is a routine currently being handling the redeem
				if _, ok := txmgr.redeemWaitingSignature.Load(redeem.RequestTxHash); ok {
					continue
				}

				ok, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
				if err != nil {
					logger1.Error("Etherman: failed to check if prepared")
					return err
				}

				if ok {
					continue
				}

				// lock the request tx hash to prevent multiple routines from handling
				// the same redeem
				txmgr.redeemWaitingSignature.Store(redeem.RequestTxHash, true)

				// start a go routine to handle the redeem
				go func() {
					// release the lock on the request tx hash due to failure
					// to get outpoints
					defer txmgr.redeemWaitingSignature.Delete(redeem.RequestTxHash)

					// request spendable outpoints from btc wallet
					ch1 := make(chan []eth2btcstate.Outpoint, 1)
					err := txmgr.btcWallet.Request(
						redeem.RequestTxHash,
						redeem.Amount,
						ch1,
					)
					if err != nil {
						logger1.Errorf("failed to request spendable outpoints with err=%v", err)
						return
					}

					logger1.Debug("waiting for outpoints...")
					outpoints, err := txmgr.waitforOutpoints(ctx, ch1)
					if err != nil {
						switch err {
						case context.Canceled:
							logger1.Debug("context canceled")
							return
						case context.DeadlineExceeded:
							logger1.Debug("timedout on waiting for outpoints")
							return
						case ErrEmptyOutpointsReturned:
							logger1.Debug("empty outpoints returned")
							return
						default:
							panic(err)
						}
					}

					// Compute the signing hash
					redeem.Outpoints = append([]eth2btcstate.Outpoint{}, outpoints...)
					params := createPrepareParams(redeem)
					signingHash := params.SigningHash()

					// request signature
					ch2 := make(chan *SignatureRequest, 1)
					txmgr.schnorrWallet.Sign(
						&SignatureRequest{
							RequestTxHash: redeem.RequestTxHash,
							SigningHash:   signingHash,
						},
						ch2,
					)

					logger2 := logger1.WithFields("signingHash", signingHash.String())
					logger2.Debug("waiting for signature...")

					// wait for the signature to be sent by the schnorr wallet
					req, err := txmgr.waitForSignature(ctx, signingHash, ch2)
					if err != nil {
						switch err {
						case context.DeadlineExceeded:
							logger2.Debug("timedout on waiting for signature")
							return
						case ErrInvalidSchnorrSignature:
							logger2.Debugf("received invalid schnorr signature: rx=0x%x, s=0x%x", req.Rx, req.S)
							return
						case context.Canceled:
							logger2.Debug("context canceled")
							return
						default:
							panic(err)
						}
					}

					err = txmgr.mgrdb.insertSignatureRequest(req)
					if err != nil {
						logger2.Error("failed to insert signature request in db")
						return
					}

					params.Rx = common.BigIntClone(req.Rx)
					params.S = common.BigIntClone(req.S)
					txmgr.handleRequestedSignature(params, logger2)
				}()
			}
		}
	}
}

func (txmgr *EthTxManager) Close() {
	if txmgr.mgrdb != nil {
		txmgr.mgrdb.Close()
	}
}

func (txmgr *EthTxManager) waitforOutpoints(
	ctx context.Context,
	ch <-chan []eth2btcstate.Outpoint,
) ([]eth2btcstate.Outpoint, error) {
	newCtx, cancel := context.WithTimeout(ctx, txmgr.cfg.TimeoutOnWaitingForOutpoints)
	defer cancel()

	for {
		select {
		case <-newCtx.Done():
			return nil, newCtx.Err()
		case outpoints := <-ch:
			if len(outpoints) == 0 {
				return nil, ErrEmptyOutpointsReturned
			}
			return outpoints, nil
		}
	}
}

func (txmgr *EthTxManager) waitForSignature(
	ctx context.Context,
	signingHash [32]byte,
	ch <-chan *SignatureRequest,
) (*SignatureRequest, error) {
	newCtx, cancel := context.WithTimeout(ctx, txmgr.cfg.TimeoutOnWaitingForSignature)
	defer cancel()

	for {
		select {
		case <-newCtx.Done():
			return nil, ctx.Err()
		case req := <-ch:
			if ok := common.Verify(txmgr.pubKey[:], signingHash[:], req.Rx, req.S); !ok {
				return req, ErrInvalidSchnorrSignature
			}
			return req, nil
		}
	}
}

func (txmgr *EthTxManager) handleRequestedSignature(
	params *etherman.PrepareParams,
	logger *logger.Logger,
) error {
	// Get the latest block
	latestBlock, err := txmgr.etherman.Client().BlockByNumber(context.Background(), nil)
	if err != nil {
		logger.Error("failed to get latest block")
		return err
	}
	logger.Debugf("latest block: num=%d", latestBlock.Number())

	// send the tx to prepare the requested redeem
	tx, err := txmgr.etherman.RedeemPrepare(params)
	if err != nil {
		logger.Error("failed to prepare redeem")
		return err
	}

	logger1 := logger.WithFields("prepareTx", tx.Hash().String())
	logger1.Debug("tx sent to prepare redeem")

	if latestBlock.Time() > uint64(tx.Time().Unix()) {
		logger1.Debugf("local time is behind the block time: local=%d, block=%d", tx.Time(), latestBlock.Time())
	}

	mt := &monitoredTx{
		TxHash:        tx.Hash(),
		RequestTxHash: params.RequestTxHash,
		SentAt:        latestBlock.Hash(),
		MinedAt:       [32]byte{},
	}
	err = txmgr.mgrdb.insertMonitoredTx(mt)
	if err != nil {
		logger1.Error("failed to insert monitored tx in db")
		return err
	}
	logger1.Debugf("inserted tx for monitoring after blk=0x%x", mt.SentAt)

	return nil
}

func createPrepareParams(redeem *eth2btcstate.Redeem) *etherman.PrepareParams {
	outpointTxIds := []ethcommon.Hash{}
	outpointIdxs := []uint16{}

	for _, outpoint := range redeem.Outpoints {
		outpointTxIds = append(outpointTxIds, outpoint.TxId)
		outpointIdxs = append(outpointIdxs, outpoint.Idx)
	}

	return &etherman.PrepareParams{
		RequestTxHash: redeem.RequestTxHash,
		Requester:     redeem.Requester,
		Receiver:      redeem.Receiver,
		Amount:        common.BigIntClone(redeem.Amount),
		OutpointTxIds: outpointTxIds,
		OutpointIdxs:  outpointIdxs,
	}
}
