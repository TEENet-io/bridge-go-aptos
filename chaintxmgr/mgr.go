package chaintxmgr

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/chaintxmgrdb"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	logger "github.com/sirupsen/logrus"
)

type ChainTxMgrConfig struct {
	// Loop's main interval
	IntervalCheckTime time.Duration

	// Timeout on waiting for a schnorr threshold signature
	TimeoutOnWaitingForSignature time.Duration

	// Timeout on waiting for the spendable outpoints from BTC wallet
	TimeoutOnWaitingForOutpoints time.Duration

	// Timeout Ledger version number (or block number)
	TimeoutTxLedgerNumber *big.Int
}

type ChainTxMgr struct {
	cfg              *ChainTxMgrConfig
	statedb          *state.StateDB               // concret object, (shall change to interface)
	mgrdb            chaintxmgrdb.ChainTxMgrDB    // interface
	schnorrParty     agreement.SchnorrAsyncSigner // interface
	btcUTXOResponder agreement.BtcUTXOResponder   // interface

	// Public Key for Schnorr signature verification
	pubKey [32]byte // public key is of 32 byte

	// Worker on the chain (do the interaction)
	chainWorker MgrWorker
}

// The Big Loop!
func (ctm *ChainTxMgr) Loop(ctx context.Context) error {
	logger.Debug("starting eth tx manager")
	defer logger.Debug("stopping eth tx manager")

	tickerInterval := time.NewTicker(ctm.cfg.IntervalCheckTime)
	defer tickerInterval.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tickerInterval.C:
			// do the mint
			// do the redeemPrepare
			// do the Tx status
		}
	}
}

// Find "new" mints from state db
func (ctm *ChainTxMgr) QueryMints() ([]*state.Mint, error) {
	mints, err := ctm.statedb.GetUnMinted()
	if err != nil {
		logger.Errorf("failed to get unminted: err=%v", err)
		return nil, err
	}
	return mints, nil
}

// Filter mints, drop off those already minted (tracked in mgr db)
func (cmt *ChainTxMgr) FilterMints(mints []*state.Mint) ([]*state.Mint, error) {
	_mints := []*state.Mint{}
	for _, mint := range mints {
		_refId := mint.BtcTxId.Bytes()
		_hits, err := cmt.mgrdb.GetMonitoredTxByRefIdentifier(_refId)
		if err != nil {
			logger.Errorf("failed to get monitored tx by ref id: err=%v", err)
			continue
		}
		if len(_hits) == 0 {
			_mints = append(_mints, mint)
		}
	}
	return _mints, nil
}

// Double check the mint status on chain, drop off those already minted (success minted on chain)
func (cmt *ChainTxMgr) IsDoubleMint(mint *state.Mint) (bool, error) {
	// Check if the mint is already minted on chain
	found, err := cmt.chainWorker.IsMinted(mint.BtcTxId)
	if err != nil {
		logger.Errorf("failed to check if minted: err=%v", err)
		return false, err
	}
	if found {
		logger.Debug("already minted, skip minting")
		return true, nil
	} else {
		return false, nil
	}
}

// Prepare mint params (this step contains the schnorr signature request)
func (cmt *ChainTxMgr) PrepareMint(ctx context.Context, mint *state.Mint) (*agreement.MintParameter, error) {
	mp := &agreement.MintParameter{
		BtcTxId:  mint.BtcTxId,
		Receiver: mint.Receiver,
		Amount:   common.BigIntClone(mint.Amount),
	}
	msgHash := mp.GenerateSigningHash()

	// request signature from schnorr signer
	_channel := make(chan *agreement.SignatureRequest, 1)
	err := cmt.schnorrParty.SignAsync(
		&agreement.SignatureRequest{
			Id:          mint.BtcTxId,
			SigningHash: msgHash,
		},
		_channel,
	)
	if err != nil {
		logger.Errorf("failed to request signature with err=%v", err)
		return nil, err
	}

	// wait for the signature to be sent by the schnorr wallet
	req, err := cmt.waitAndVerifySignature(ctx, msgHash, _channel)
	if err != nil {
		return nil, err
	}
	logger.Info("schnorr signature requested & received")

	// set outpoints before saving
	mp.Rx = common.BigIntClone(req.Rx)
	mp.S = common.BigIntClone(req.S)

	return mp, nil
}

func (cmt *ChainTxMgr) CallMint() {}

func (cmt *ChainTxMgr) SetMintToBeMonitored() {}

// Set a Tx to be monitored in mgr db
func (cmt *ChainTxMgr) setMonitoredTx(tx chaintxmgrdb.MonitoredTx) error {
	return cmt.mgrdb.InsertMonitoredTx(tx)
}

// Wait for Schnorr Signature.
// Then verify Signature
func (cmt *ChainTxMgr) waitAndVerifySignature(
	ctx context.Context,
	signingHash [32]byte,
	ch <-chan *agreement.SignatureRequest,
) (*agreement.SignatureRequest, error) {
	newCtx, cancel := context.WithTimeout(ctx, cmt.cfg.TimeoutOnWaitingForSignature)
	defer cancel()

	for {
		select {
		case <-newCtx.Done():
			return nil, ctx.Err()
		case req := <-ch:
			if ok := common.Verify(cmt.pubKey[:], signingHash[:], req.Rx, req.S); !ok {
				return req, fmt.Errorf("ERR_BAD_SIGNATURE: signature verification failed")
			}
			return req, nil
		}
	}
}

// Mgr's worker on chain, do the dirty job.
type MgrWorker interface {
	// Verify if the mint is already minted on chain
	// The query uses the mint's BTC tx id (prevent double mint check)
	IsMinted(btcTxId [32]byte) (bool, error)
}
