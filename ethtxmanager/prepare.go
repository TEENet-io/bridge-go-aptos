package ethtxmanager

import (
	"context"
	"errors"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	ErrIsPrepared                         = errors.New("failed to call bridge.IsPrepared")
	ErrGetMonitoredTxByRequestTxHash      = errors.New("failed to get monitored tx by request tx hash")
	ErrGetSignatureRequestByRequestTxHash = errors.New("failed to get signature request by request tx hash")
	ErrBtcWalletRequest                   = errors.New("failed to request spendable outpoints from btc wallet")
	ErrSchnorrWalletSign                  = errors.New("failed to request signature from schnorr wallet")
	ErrInsertSignatureRequest             = errors.New("failed to insert signature request in db")
	ErrRedeemPrepare                      = errors.New("failed to call bridge.RedeemPrepare")
	ErrInsertMonitoredTx                  = errors.New("failed to insert monitored tx in db")
)

func (txmgr *EthTxManager) prepareRedeem(ctx context.Context, redeem *state.Redeem) error {
	// lock the request tx hash to prevent multiple routines from handling
	// the same redeem when entering.
	txmgr.redeemLock.Store(redeem.RequestTxHash, true)
	defer txmgr.redeemLock.Delete(redeem.RequestTxHash)

	newLogger := logger.WithFields("requestTxHash", redeem.RequestTxHash.String())

	// Check the redeem status on bridge contract. If false, either the redeem
	// has been handled or there is a pending tx that tries to prepare the redeem.
	ok, err := txmgr.etherman.IsPrepared(redeem.RequestTxHash)
	if err != nil {
		newLogger.Errorf("Etherman: failed to check if prepared: err=%v", err)
		return ErrIsPrepared
	}
	if ok {
		return nil
	}

	// Check whether there is any pending tx that is preparing the redeem
	_, ok, err = txmgr.mgrdb.GetMonitoredTxByRequestTxHash(redeem.RequestTxHash)
	if err != nil {
		newLogger.Errorf("failed to get monitored tx by request tx hash: err=%v", err)
		return ErrGetMonitoredTxByRequestTxHash
	}
	if ok {
		return nil
	}

	// It is possible that the request for signature is successful but the
	// redeem prepare tx is failed. In this case, we should resend a tx
	// right away to prepare the redeem.
	_, ok, err = txmgr.mgrdb.GetSignatureRequestByRequestTxHash(redeem.RequestTxHash)
	if err != nil {
		newLogger.Error("failed to get signature request by request tx hash")
		return ErrGetSignatureRequestByRequestTxHash
	}
	if ok {
		newLogger.Debug("signature request already exists, resend a tx for preparing redeem")
		return txmgr.handleRequestedSignature(createPrepareParams(redeem), newLogger)
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
	newLogger.Debug("outpoints received")

	// Compute the signing hash
	redeem.Outpoints = append([]state.Outpoint{}, outpoints...)
	params := createPrepareParams(redeem)
	signingHash := params.SigningHash()

	// request signature
	chForSignature := make(chan *SignatureRequest, 1)
	err = txmgr.schnorrWallet.Sign(
		&SignatureRequest{
			RequestTxHash: redeem.RequestTxHash,
			SigningHash:   signingHash,
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

	err = txmgr.mgrdb.insertSignatureRequest(req)
	if err != nil {
		newLogger.Errorf("failed to insert signature request in db: err=%v", err)
		return ErrInsertSignatureRequest
	}
	newLogger.Debug("inserted signature request in db")

	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)
	return txmgr.handleRequestedSignature(params, newLogger)
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

func (txmgr *EthTxManager) handleRequestedSignature(
	params *etherman.PrepareParams,
	logger *logger.Logger,
) error {
	// Get the latest block
	latest, err := txmgr.etherman.Client().HeaderByNumber(context.Background(), nil)
	if err != nil {
		logger.Errorf("failed to get latest block: err=%v", err)
		return ErrHeaderByNumber
	}
	logger.Debugf("got latest block: num=%d", latest.Number)

	// Send the tx to prepare the requested redeem
	tx, err := txmgr.etherman.RedeemPrepare(params)
	if err != nil {
		logger.Errorf("failed to prepare redeem, err=%v", err)
		return ErrRedeemPrepare
	}

	logger1 := logger.WithFields("prepareTx", tx.Hash().String())
	logger1.Debug("tx sent to prepare redeem")

	mt := &monitoredTx{
		TxHash:        tx.Hash(),
		RequestTxHash: params.RequestTxHash,
		SentAfter:     latest.Hash(),
	}
	err = txmgr.mgrdb.insertMonitoredTx(mt)
	if err != nil {
		logger1.Errorf("failed to insert monitored tx in db, err=%v", err)
		return ErrInsertMonitoredTx
	}
	logger1.Debugf("inserted monitored tx: sentAfter=0x%x", mt.SentAfter)

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
