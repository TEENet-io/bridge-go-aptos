package ethtxmanager

import (
	"context"
	"errors"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
	logger "github.com/sirupsen/logrus"
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

	newLogger := logger.WithField("btcTxId", mint.BtcTxId.String())

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

	// Compute the signing hash
	params := &etherman.MintParams{
		BtcTxId:  mint.BtcTxId,
		Receiver: mint.Receiver,
		Amount:   common.BigIntClone(mint.Amount),
	}
	signingHash := params.SigningHash()

	// request signature
	chForSignature := make(chan *agreement.SignatureRequest, 1)
	err = txmgr.schnorrWallet.SignAsync(
		&agreement.SignatureRequest{
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
	newLogger.Info("schnorr signature requested & received")

	// set outpoints before saving
	params.Rx = common.BigIntClone(req.Rx)
	params.S = common.BigIntClone(req.S)

	return txmgr.createMintTx(params, newLogger)
}

// Create Mint Tx, then Send Mint Tx, Then Insert Mint Tx in mgr db
func (txmgr *EthTxManager) createMintTx(
	params *etherman.MintParams,
	logger *logger.Entry,
) error {
	// Get the latest_header block
	latest_header, err := txmgr.etherman.Client().HeaderByNumber(context.Background(), nil)
	if err != nil {
		logger.Errorf("failed to get latest block header: err=%v", err)
		return ErrEthermanHeaderByNumber
	}
	logger.Debugf("got latest eth block: num=%d", latest_header.Number)

	// Send the real Mint Tx to Ethereum
	tx, err := txmgr.etherman.Mint(params)
	if err != nil {
		logger.Errorf("failed to mint: err=%v", err)
		return ErrEthermanMint
	}

	// logger.WithField("mintTx", tx.Hash().String()).Info("Mint tx sent")

	// Save the monitored tx
	mt := &MonitoredTx{
		TxHash:        tx.Hash(),
		RefIdentifier: params.BtcTxId,
		SentAfter:     latest_header.Hash(), // interesting, using block hash as sendAfter not block number.
		SentAfterBlk:  latest_header.Number.Int64(),
	}
	logger.WithField("hash", latest_header.Hash()).WithField("num", latest_header.Number).Debug("latest block (eth)")
	err = txmgr.mgrdb.InsertPendingMonitoredTx(mt)
	if err != nil {
		logger.Errorf("failed to insert pending monitored tx: err=%v", err)
		return ErrDBOpInsertMonitoredTx
	}
	logger.Debugf("inserted monitored tx: sentAfter(last eth block hash)=0x%x, sentAfterBlk=%d", mt.SentAfter, mt.SentAfterBlk)

	return nil
}
