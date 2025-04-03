package chaintxmgr

import (
	"context"
	"fmt"
	"math/big"
	"sync"
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
	pubKey           [32]byte                     // // Public Key for Schnorr signature verification, 32 byte

	chainWorker MgrWorker // Chain Worker (do the interaction with chain)

	mgrdbLock  sync.Mutex // Prevent race condition, both read/write lock
	mintLock   sync.Mutex // Prevent race condition
	redeemLock sync.Map   // Prevent race condition
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

			// do the mint procedure
			mint_err := ctm.procedureMint(ctx)
			if mint_err != nil {
				logger.Errorf("failed to process mint: err=%v", mint_err)
			}
			// do the redeemPrepare
			// do the Tx status
		}
	}
}

func (ctm *ChainTxMgr) procedureMint(ctx context.Context) error {
	// 0. Aquire necessary locks
	// 1. Find mints from state db
	// 2. Filter those already minted (tracked in mgr db)
	// 3. for each mint, Check if the mint is already minted on chain
	// 4. for each mint, Prepare mint params (this step contains the schnorr signature request)
	// 5. for each mint, Call mint() function on chain
	// 6. for each mint, Set Tx to the mgrdb, and it is "pending" status

	// 0. Aquire necessary locks
	ctm.mgrdbLock.Lock()
	defer ctm.mgrdbLock.Unlock()

	ctm.mintLock.Lock()
	defer ctm.mintLock.Unlock()

	// 1. Find mints from state db
	mints, err := ctm.QueryMints()
	if err != nil {
		logger.Errorf("failed to query mints: err=%v", err)
		return err
	}

	// 2. Filter those already minted (tracked in mgr db)
	mints_clean, err := ctm.FilterMints(mints)
	if err != nil {
		logger.Errorf("failed to filter mints against mgrdb: err=%v", err)
		return err
	}

	if len(mints_clean) == 0 {
		logger.Debug("no new mints to process")
		return nil
	}

	for _, mint := range mints_clean {
		// 3. Check if the mint is already minted on chain
		found, err := ctm.IsDoubleMint(mint)
		if err != nil {
			logger.Errorf("failed to check if minted: err=%v", err)
			continue
		}
		if found {
			continue
		}

		// 4. Prepare mint params (this step contains the schnorr signature request)
		_mint_params, err := ctm.PrepareMint(ctx, mint)
		if err != nil {
			logger.Errorf("failed to prepare mint params: err=%v", err)
			continue
		}

		// 5. Call mint() function on chain
		tx_id, ledger_number, err := ctm.CallMint(_mint_params)
		if err != nil {
			logger.Errorf("failed to call mint() on chain: err=%v", err)
			continue
		}

		// 6. Set Tx to the mgrdb, and it is "pending" status
		err = ctm.SetMintToBeMonitored(_mint_params, tx_id, ledger_number, chaintxmgrdb.Pending)
		if err != nil {
			logger.Errorf("failed to set mint to be monitored in mgrdb: err=%v", err)
			continue
		}
	}
	return nil
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

// Call the actual mint() function on chain
// Return the (tx_id, error)
func (cmt *ChainTxMgr) CallMint(mp *agreement.MintParameter) ([]byte, *big.Int, error) {
	// Send the real Mint Tx to Ethereum
	tx_id, ledger_number, err := cmt.chainWorker.DoMint(mp)
	if err != nil {
		logger.Errorf("failed to call mint() on chain: err=%v", err)
		return nil, nil, err
	}

	logger.WithField("mintTx", common.ByteSliceToPureHexStr(tx_id)).Info("Mint tx sent")
	return tx_id, ledger_number, nil
}

// Set Tx to the mgrdb, and it is "pending" status
// TODO: Tx may actually be "limbo" status after success submission.
// TODO: Need a second around of scan to change it from "limbo" to "pending".
func (cmt *ChainTxMgr) SetMintToBeMonitored(mp *agreement.MintParameter, txId []byte, sentAt *big.Int, status chaintxmgrdb.MonitoredTxStatus) error {
	// Set the mint to be monitored in mgr db
	monitoredTx := chaintxmgrdb.MonitoredTx{
		TxIdentifier:                txId,
		RefIdentifier:               mp.BtcTxId.Bytes(),
		SentBlockchainLedgerNumber:  common.BigIntClone(sentAt),
		FoundBlockchainLedgerNumber: nil,
		TxStatus:                    status,
	}
	return cmt.setMonitoredTx(&monitoredTx)
}

// Set a Tx to be monitored in mgr db
func (cmt *ChainTxMgr) setMonitoredTx(tx *chaintxmgrdb.MonitoredTx) error {
	return cmt.mgrdb.InsertMonitoredTx(tx)
}

// Wait for Schnorr Signature.
// Then verify Signature.
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
	// Get the latest ledger number from chain (block number on eth, ledger version number on aptos)
	// This number marks the latest height (advancement) of blockchain.
	GetLatestLedgerNumber() (*big.Int, error)

	// Call the smart contract and verify if the mint is already minted on chain
	// The query uses the mint's BTC tx id (prevent double mint check)
	IsMinted(btcTxId [32]byte) (bool, error)

	// Call the actual mint() on smart contract on chain
	// Note: this function shall return the approximate ledger number when this tx is submitted to blockchain.
	// If ledger number field is unknown, set to nil.
	// Return the (mint_tx_hash, sent_at_ledger_number, error)
	DoMint(mint *agreement.MintParameter) ([]byte, *big.Int, error)
}
