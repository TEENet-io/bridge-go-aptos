package ethtxmanager

import (
	"context"
	"errors"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
)

var (
	ErrBridgeIsMinted = errors.New("failed to call bridge.IsMinted")
	ErrEthermanMint   = errors.New("failed to call etherman.Mint")
)

func (txmgr *EthTxManager) mint(ctx context.Context, mint *state.Mint) error {
	// lock btcTxId hash to prevent multiple routines from handling
	// the same mint
	txmgr.mintLock.Store(mint.BtcTxId, true)
	defer txmgr.mintLock.Delete(mint.BtcTxId)

	newLogger := logger.WithFields("btcTxId", mint.BtcTxId.String())

	// Check the mint status on bridge contract.
	ok, err := txmgr.etherman.IsMinted(mint.BtcTxId)
	if err != nil {
		newLogger.Errorf("Etherman: failed to check if minted: err=%v", err)
		return ErrBridgeIsMinted
	}
	if ok {
		newLogger.Debug("already minted, skip minting")
		return nil
	}

	// request spendable outpoints from btc wallet
	chForOutpoints := make(chan []state.Outpoint, 1)
	err = txmgr.btcWallet.Request(
		mint.BtcTxId,
		mint.Amount,
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
	params := &etherman.MintParams{
		BtcTxId:  mint.BtcTxId,
		Receiver: mint.Receiver,
		Amount:   common.BigIntClone(mint.Amount),
	}
	signingHash := params.SigningHash()

	// request signature
	chForSignature := make(chan *SignatureRequest, 1)
	err = txmgr.schnorrWallet.Sign(
		&SignatureRequest{
			Id:          mint.BtcTxId,
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

	// set outpoints before saving
	req.Outpoints = append([]state.Outpoint{}, outpoints...)
	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)

	return txmgr.handleMintTx(params, req, newLogger)
}

func (txmgr *EthTxManager) handleMintTx(
	params *etherman.MintParams,
	req *SignatureRequest,
	logger *logger.Logger,
) error {
	// Get the latest block
	latest, err := txmgr.etherman.Client().HeaderByNumber(context.Background(), nil)
	if err != nil {
		logger.Errorf("failed to get latest block: err=%v", err)
		return ErrEthermanHeaderByNumber
	}
	logger.Debugf("got latest block: num=%d", latest.Number)

	tx, err := txmgr.etherman.Mint(params)
	if err != nil {
		logger.Errorf("failed to mint: err=%v", err)
		return ErrEthermanMint
	}

	newLogger := logger.WithFields("mintTx", tx.Hash().String())
	newLogger.Debugf("mint tx sent: tx=%s", tx.Hash().String())

	// Save the monitored tx
	mt := &MonitoredTx{
		TxHash:      tx.Hash(),
		Id:          params.BtcTxId,
		SigningHash: req.SigningHash,
		Outpoints:   append([]state.Outpoint{}, req.Outpoints...),
		Rx:          common.BigIntClone(req.Rx),
		S:           common.BigIntClone(req.S),
		SentAfter:   latest.Hash(),
	}
	err = txmgr.mgrdb.InsertPendingMonitoredTx(mt)
	if err != nil {
		logger.Errorf("failed to insert pending monitored tx: err=%v", err)
		return ErrDBOpInsertMonitoredTx
	}
	newLogger.Debugf("inserted monitored tx: sentAfter=0x%x", mt.SentAfter)

	return nil
}
