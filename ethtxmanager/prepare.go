package ethtxmanager

import (
	"context"
	"errors"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
)

var (
	ErrBridgeIsPrepared                       = errors.New("failed to call bridge.IsPrepared")
	ErrDBOpGetMonitoredTxByID                 = errors.New("failed to get monitored tx by id")
	ErrDBOpGetSignatureRequestByRequestTxHash = errors.New("failed to get signature request by request tx hash")
	ErrBtcWalletRequest                       = errors.New("failed to request spendable outpoints from btc wallet")
	ErrSchnorrWalletSign                      = errors.New("failed to request signature from schnorr wallet")
	ErrDBOpInsertSignatureRequest             = errors.New("failed to insert signature request in db")
	ErrBridgeRedeemPrepare                    = errors.New("failed to call bridge.RedeemPrepare")
	ErrDBOpInsertMonitoredTx                  = errors.New("failed to insert monitored tx in db")
	ErrEmptyOutpointsReturned                 = errors.New("empty outpoints returned")
	ErrInvalidSchnorrSignature                = errors.New("invalid schnorr signature")
)

// prepareRedeem select UTXOs to satisfy the redeem.
func (txmgr *EthTxManager) prepareRedeem(ctx context.Context, redeem *state.Redeem) (*ethcommon.Hash, error) {
	// lock the request tx hash to prevent multiple routines from handling
	// the same redeem when entering.
	txmgr.redeemLock.Store(redeem.RequestTxHash, true)
	defer txmgr.redeemLock.Delete(redeem.RequestTxHash)

	newLogger := logger.WithField("reqTx", redeem.RequestTxHash.String())

	// Check the redeem status on bridge contract. If false, either the redeem
	// has been handled or there is a pending tx that tries to prepare the redeem.
	found, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
	if err != nil {
		newLogger.Errorf("Etherman: failed to check if prepared: err=%v", err)
		return nil, ErrBridgeIsPrepared
	}
	if found {
		newLogger.Debug("redeem already prepared, skip")
		return &redeem.PrepareTxHash, nil
	} else {
		newLogger.Info("redeem not prepared, start to prepare")
	}

	// request spendable outpoints from btc wallet
	chForOutpoints := make(chan []state.Outpoint, 1)
	err = txmgr.btcWallet.Request(
		redeem.RequestTxHash,
		redeem.Amount,
		chForOutpoints,
	)
	if err != nil {
		newLogger.WithField("err", err).Error("failed to request spendable outpoints")
		return nil, ErrBtcWalletRequest
	}

	outpoints, err := txmgr.waitforOutpoints(ctx, chForOutpoints)
	if err != nil {
		return nil, err
	}
	newLogger.WithField("num", len(outpoints)).Info("outpoints received")

	// Compute the signing hash
	redeem.Outpoints = append([]state.Outpoint{}, outpoints...)
	params := createPrepareParams(redeem)
	signingHash := params.SigningHash()

	// request signature
	chForSignature := make(chan *SignatureRequest, 1)
	err = txmgr.schnorrWallet.SignAsync(
		&SignatureRequest{
			Id:          redeem.RequestTxHash,
			SigningHash: signingHash,
		},
		chForSignature,
	)
	if err != nil {
		newLogger.WithField("err", err).Error("failed to request signature")
		return nil, ErrSchnorrWalletSign
	}

	// wait for the signature to be sent by the schnorr wallet
	req, err := txmgr.waitForSignature(ctx, signingHash, chForSignature)
	if err != nil {
		return nil, err
	}
	newLogger.Info("schnorr signature received")

	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)
	return txmgr.createRedeemPrepareTx(params, newLogger)
}

func (txmgr *EthTxManager) waitforOutpoints(
	ctx context.Context,
	ch <-chan []state.Outpoint,
) ([]state.Outpoint, error) {
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

// Return the *(redeem prepare tx Hash) + error
func (txmgr *EthTxManager) createRedeemPrepareTx(
	params *etherman.PrepareParams,
	logger *logger.Entry,
) (*ethcommon.Hash, error) {
	// Get the latest block
	latest, err := txmgr.etherman.Client().HeaderByNumber(context.Background(), nil)
	if err != nil {
		logger.WithField("err", err).Error("failed to get latest block")
		return nil, ErrEthermanHeaderByNumber
	}
	logger.WithField("latestBlockNumber", latest.Number).Debug("latest block")

	// Send the redeemprepare Tx on ETH
	tx, err := txmgr.etherman.RedeemPrepare(params)
	if err != nil {
		logger.WithField("err", err).Error("failed to send RedeemPrepare tx")
		return nil, ErrBridgeRedeemPrepare
	}

	newLogger := logger.WithField("prepareTx", tx.Hash().String())
	newLogger.Info("prepare redeem tx sent")

	mt := &MonitoredTx{
		TxHash:       tx.Hash(),
		Id:           params.RequestTxHash,
		SentAfter:    latest.Hash(),
		SentAfterBlk: latest.Number.Int64(),
	}
	err = txmgr.mgrdb.InsertPendingMonitoredTx(mt)
	if err != nil {
		newLogger.Errorf("failed to insert monitored tx in db, err=%v", err)
		return nil, ErrDBOpInsertMonitoredTx
	}
	newLogger.Debugf("inserted monitored tx: sentAfter=0x%x", mt.SentAfter)

	_hash := tx.Hash()
	return &_hash, nil
}

func createPrepareParams(redeem *state.Redeem) *etherman.PrepareParams {
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
