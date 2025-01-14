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

func (txmgr *EthTxManager) prepareRedeem(ctx context.Context, redeem *state.Redeem) error {
	// lock the request tx hash to prevent multiple routines from handling
	// the same redeem when entering.
	txmgr.redeemLock.Store(redeem.RequestTxHash, true)
	defer txmgr.redeemLock.Delete(redeem.RequestTxHash)

	newLogger := logger.WithField("requestTxHash", redeem.RequestTxHash.String())

	// Check the redeem status on bridge contract. If false, either the redeem
	// has been handled or there is a pending tx that tries to prepare the redeem.
	ok, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
	if err != nil {
		newLogger.Errorf("Etherman: failed to check if prepared: err=%v", err)
		return ErrBridgeIsPrepared
	}
	if ok {
		newLogger.Debug("redeem already prepared, skip preparing redeem")
		return nil
	}

	// request spendable outpoints from btc wallet
	chForOutpoints := make(chan []state.Outpoint, 1)
	err = txmgr.btcWallet.Request(
		redeem.RequestTxHash,
		redeem.Amount,
		chForOutpoints,
	)
	if err != nil {
		newLogger.Errorf("failed to request spendable outpoints with err=%v", err)
		return ErrBtcWalletRequest
	}

	outpoints, err := txmgr.waitforOutpoints(ctx, chForOutpoints)
	if err != nil {
		return err
	}
	newLogger.Debugf("outpoints received: %d", len(outpoints))

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
		newLogger.Errorf("failed to request signature with err=%v", err)
		return ErrSchnorrWalletSign
	}

	// wait for the signature to be sent by the schnorr wallet
	req, err := txmgr.waitForSignature(ctx, signingHash, chForSignature)
	if err != nil {
		return err
	}
	newLogger.Debug("schnorr signature received")

	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)
	return txmgr.handleRedeemPrepareTx(params, newLogger)
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

func (txmgr *EthTxManager) handleRedeemPrepareTx(
	params *etherman.PrepareParams,
	logger *logger.Entry,
) error {
	// Get the latest block
	latest, err := txmgr.etherman.Client().HeaderByNumber(context.Background(), nil)
	if err != nil {
		logger.Errorf("failed to get latest block: err=%v", err)
		return ErrEthermanHeaderByNumber
	}
	logger.WithField("latestBlockNumber", latest.Number).Debug("latest block")

	// Send the tx to prepare the requested redeem
	tx, err := txmgr.etherman.RedeemPrepare(params)
	if err != nil {
		logger.Errorf("failed to send tx, err=%v", err)
		return ErrBridgeRedeemPrepare
	}

	newLogger := logger.WithField("prepareTx", tx.Hash().String())
	newLogger.Debug("tx sent to prepare redeem")

	mt := &MonitoredTx{
		TxHash:    tx.Hash(),
		Id:        params.RequestTxHash,
		SentAfter: latest.Hash(),
	}
	err = txmgr.mgrdb.InsertPendingMonitoredTx(mt)
	if err != nil {
		newLogger.Errorf("failed to insert monitored tx in db, err=%v", err)
		return ErrDBOpInsertMonitoredTx
	}
	newLogger.Debugf("inserted monitored tx: sentAfter=0x%x", mt.SentAfter)

	return nil
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
