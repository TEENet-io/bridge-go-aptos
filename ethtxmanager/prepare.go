package ethtxmanager

import (
	"context"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func (txmgr *EthTxManager) prepareRedeem(ctx context.Context, redeem *eth2btcstate.Redeem, logger *logger.Logger) {
	// lock the request tx hash to prevent multiple routines from handling
	// the same redeem
	txmgr.redeemLock.Store(redeem.RequestTxHash, true)

	// release the lock on the request tx hash due to failure
	// to get outpoints
	defer txmgr.redeemLock.Delete(redeem.RequestTxHash)

	// It is possible that the request for signature is successful but the
	// redeem prepare tx is failed. In this case, we should resend a tx
	// right away to prepare the redeem.
	_, ok, err := txmgr.mgrdb.GetSignatureRequestByRequestTxHash(redeem.RequestTxHash)
	if err != nil {
		logger.Error("failed to get signature request by request tx hash")
		return
	}
	if ok {
		logger.Debug("signature request already exists, resend a tx for preparing redeem")
		txmgr.handleRequestedSignature(createPrepareParams(redeem), logger)
		return
	}

	// request spendable outpoints from btc wallet
	chForOutpoints := make(chan []eth2btcstate.Outpoint, 1)
	err = txmgr.btcWallet.Request(
		redeem.RequestTxHash,
		redeem.Amount,
		chForOutpoints,
	)
	if err != nil {
		logger.Errorf("failed to request spendable outpoints with err=%v", err)
		return
	}

	logger.Debug("waiting for outpoints...")
	outpoints, err := txmgr.waitforOutpoints(ctx, chForOutpoints)
	if err != nil {
		switch err {
		case context.Canceled:
			logger.Debug("context canceled")
			return
		case context.DeadlineExceeded:
			logger.Debug("timedout on waiting for outpoints")
			return
		case ErrEmptyOutpointsReturned:
			logger.Debug("empty outpoints returned")
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
	chForSignature := make(chan *SignatureRequest, 1)
	txmgr.schnorrWallet.Sign(
		&SignatureRequest{
			RequestTxHash: redeem.RequestTxHash,
			SigningHash:   signingHash,
		},
		chForSignature,
	)

	logger.Debug("waiting for signature...")

	// wait for the signature to be sent by the schnorr wallet
	req, err := txmgr.waitForSignature(ctx, signingHash, chForSignature)
	if err != nil {
		switch err {
		case context.DeadlineExceeded:
			logger.Debug("timedout on waiting for signature")
			return
		case ErrInvalidSchnorrSignature:
			logger.Debugf("received invalid schnorr signature: rx=0x%x, s=0x%x", req.Rx, req.S)
			return
		case context.Canceled:
			logger.Debug("context canceled")
			return
		default:
			panic(err)
		}
	}

	err = txmgr.mgrdb.insertSignatureRequest(req)
	if err != nil {
		logger.Error("failed to insert signature request in db")
		return
	}

	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)
	txmgr.handleRequestedSignature(params, logger)
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
		logger.Errorf("failed to get latest block: err=%v", err)
		return err
	}
	logger.Debugf("latest block: num=%d", latestBlock.Number())

	// Send the tx to prepare the requested redeem
	tx, err := txmgr.etherman.RedeemPrepare(params)
	if err != nil {
		logger.Errorf("failed to prepare redeem, err=%v", err)
		return err
	}

	logger1 := logger.WithFields("prepareTx", tx.Hash().String())
	logger1.Debug("tx sent to prepare redeem")

	mt := &monitoredTx{
		TxHash:        tx.Hash(),
		RequestTxHash: params.RequestTxHash,
		SentAfter:     latestBlock.Hash(),
	}
	err = txmgr.mgrdb.insertMonitoredTx(mt)
	if err != nil {
		logger1.Errorf("failed to insert monitored tx in db, err=%v", err)
		return err
	}
	logger1.Debugf("inserted tx for monitoring after blk=0x%x", mt.SentAfter)

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
