// TODO: Improve the log (add params)
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
	// The Tx is considered "Timeout" after such period has passed and not mined.
	// Whether re-send or just drop the Tx is another story.
	// But we need to know and mark it clearly.
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

	mgrdbLock  sync.Mutex // Prevent race condition, both read/write lock to db.
	mintLock   sync.Mutex // Prevent race condition
	redeemLock sync.Mutex // Prevent race condition
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

			// do the redeemPrepare procedure
			prepare_err := ctm.procedurePrepare(ctx)
			if prepare_err != nil {
				logger.Errorf("failed to process prepare: err=%v", prepare_err)
			}

			// do the Tx status tracking procedure
			mark_err := ctm.procedureMarkTxStatus()
			if mark_err != nil {
				logger.Errorf("failed to process mark tx status: err=%v", mark_err)
			}
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

		// Extra: if the ledger_number is nil, we try the <best effort> to set it
		if ledger_number == nil {
			latest_ledger_number, err := ctm.chainWorker.GetLatestLedgerNumber()
			if err != nil {
				logger.Errorf("failed to get latest ledger number: err=%v", err)
			} else {
				if latest_ledger_number != nil {
					ledger_number = latest_ledger_number
				} else {
					logger.Errorf("latest ledger number is nil")
				}
			}
		}

		// 6. Set Tx to the mgrdb, and it is "pending" status
		err = ctm.SetMintToBeMonitored(_mint_params, tx_id, ledger_number, agreement.Pending)
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
func (ctm *ChainTxMgr) FilterMints(mints []*state.Mint) ([]*state.Mint, error) {
	_mints := []*state.Mint{}
	for _, mint := range mints {
		_refId := mint.BtcTxId.Bytes()
		_hits, err := ctm.mgrdb.GetMonitoredTxByRefIdentifier(_refId)
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
func (ctm *ChainTxMgr) IsDoubleMint(mint *state.Mint) (bool, error) {
	// Check if the mint is already minted on chain
	found, err := ctm.chainWorker.IsMinted(mint.BtcTxId)
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
func (ctm *ChainTxMgr) PrepareMint(ctx context.Context, mint *state.Mint) (*agreement.MintParameter, error) {
	mp := &agreement.MintParameter{
		BtcTxId:  mint.BtcTxId,
		Receiver: mint.Receiver,
		Amount:   common.BigIntClone(mint.Amount),
	}
	msgHash := mp.GenerateMsgHash()

	// request signature from schnorr signer
	_channel := make(chan *agreement.SignatureRequest, 1)
	err := ctm.schnorrParty.SignAsync(
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
	req, err := ctm.waitAndVerifySignature(ctx, msgHash, _channel)
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
// Return the (tx_id, sent_at_ledger_number, error)
func (ctm *ChainTxMgr) CallMint(mp *agreement.MintParameter) ([]byte, *big.Int, error) {
	// Send the real Mint Tx to Ethereum
	tx_id, ledger_number, err := ctm.chainWorker.DoMint(mp)
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
func (ctm *ChainTxMgr) SetMintToBeMonitored(mp *agreement.MintParameter, txId []byte, sentAt *big.Int, status agreement.MonitoredTxStatus) error {
	// Set the mint to be monitored in mgr db
	monitoredTx := chaintxmgrdb.MonitoredTx{
		TxIdentifier:                txId,
		RefIdentifier:               mp.BtcTxId.Bytes(),
		SentBlockchainLedgerNumber:  common.BigIntClone(sentAt),
		FoundBlockchainLedgerNumber: nil,
		TxStatus:                    status,
	}
	return ctm.setMonitoredTx(&monitoredTx)
}

func (ctm *ChainTxMgr) SetPrepareToBeMonitored(pp *agreement.PrepareParameter, txId []byte, sentAt *big.Int, status agreement.MonitoredTxStatus) error {
	// Set the mint to be monitored in mgr db
	monitoredTx := chaintxmgrdb.MonitoredTx{
		TxIdentifier:                txId,
		RefIdentifier:               pp.RequestTxHash.Bytes(),
		SentBlockchainLedgerNumber:  common.BigIntClone(sentAt),
		FoundBlockchainLedgerNumber: nil,
		TxStatus:                    status,
	}
	return ctm.setMonitoredTx(&monitoredTx)
}

// Set a Tx to be monitored in mgr db
func (ctm *ChainTxMgr) setMonitoredTx(tx *chaintxmgrdb.MonitoredTx) error {
	return ctm.mgrdb.InsertMonitoredTx(tx)
}

// Find not prepared redeems from the database
func (ctm *ChainTxMgr) QueryUnPrepared() ([]*state.Redeem, error) {
	redeems, err := ctm.statedb.GetRedeemsByStatus(state.RedeemStatusRequested)
	if err != nil {
		logger.Errorf("failed to get redeems by status: err=%v", err)
		return nil, err
	}
	return redeems, nil
}

// Filter out those redeems that are already prepared (tracked in mgr db)
func (ctm *ChainTxMgr) FilterUnPrepared(redeems []*state.Redeem) ([]*state.Redeem, error) {
	// Filter out those already prepared
	_unprepared := []*state.Redeem{}
	for _, redeem := range redeems {
		_refId := redeem.RequestTxHash.Bytes() // Use reqTxHash as the reference id
		_hits, err := ctm.mgrdb.GetMonitoredTxByRefIdentifier(_refId)
		if err != nil {
			logger.Errorf("failed to get monitored tx by ref id: err=%v", err)
			continue
		}
		if len(_hits) == 0 {
			_unprepared = append(_unprepared, redeem)
		}
	}
	return _unprepared, nil
}

// Double check the redeem's prepare on chain, drop off those already prepared (success on chain)
func (ctm *ChainTxMgr) IsDoublePrepare(redeem *state.Redeem) (bool, error) {
	// Check if the redeem is already prepared on chain
	found, err := ctm.chainWorker.IsPrepared(redeem.RequestTxHash)
	if err != nil {
		logger.Errorf("failed to check if is prepared: err=%v", err)
		return false, err
	}
	if found {
		logger.Debug("already prepared, skip preparing")
		return true, nil
	} else {
		return false, nil
	}
}

// Prepare the RedeemPrepare Tx needed parameters.
func (ctm *ChainTxMgr) PreparePrepare(ctx context.Context, redeem *state.Redeem) (*agreement.PrepareParameter, error) {
	// Query the BTC UTXO Responder for UTXO outpoints
	// request spendable outpoints from btc wallet
	_channel_outpoints := make(chan []agreement.BtcOutpoint, 1)
	err := ctm.btcUTXOResponder.Request(
		redeem.RequestTxHash.Bytes(),
		redeem.Amount,
		_channel_outpoints,
	)
	if err != nil {
		logger.WithField("err", err).Error("failed to request spendable UTXO outpoints")
		return nil, fmt.Errorf("ERR_BTC_UTXO_REQUEST: %v", err)
	}

	_outpoints, err := ctm.waitForOutPoints(ctx, _channel_outpoints)
	if err != nil {
		return nil, err
	}
	logger.WithField("num", len(_outpoints)).Info("UTXO outpoints received")

	// Stuff the PrepareParameter
	pp := &agreement.PrepareParameter{
		RequestTxHash: redeem.RequestTxHash,
		Requester:     redeem.Requester,
		Receiver:      redeem.Receiver,
		Amount:        common.BigIntClone(redeem.Amount),
	}
	pp.OutpointTxIds, pp.OutpointIdxs = agreement.ConvertOutpoints(_outpoints)

	// Generate msg hash before requesting signature
	msgHash := pp.GenerateMsgHash()

	// request signature from schnorr signer over the msg hash
	_channel := make(chan *agreement.SignatureRequest, 1)
	err = ctm.schnorrParty.SignAsync(
		&agreement.SignatureRequest{
			Id:          redeem.RequestTxHash,
			SigningHash: msgHash,
		},
		_channel,
	)
	if err != nil {
		logger.Errorf("failed to request signature with err=%v", err)
		return nil, err
	}
	// wait...
	req, err := ctm.waitAndVerifySignature(ctx, msgHash, _channel)
	if err != nil {
		return nil, err
	}
	logger.Info("schnorr signature requested & received")

	// Stuff the prepare parameter
	pp.Rx = common.BigIntClone(req.Rx)
	pp.S = common.BigIntClone(req.S)

	return pp, nil
}

// Call the actual prepareRedeem() function on chain
// Return the (tx_id, sent_at_ledger_number, error)
func (ctm *ChainTxMgr) CallPrepare(pp *agreement.PrepareParameter) ([]byte, *big.Int, error) {
	// Send the real Prepare Tx to Ethereum
	tx_id, ledger_number, err := ctm.chainWorker.DoPrepare(pp)
	if err != nil {
		logger.Errorf("failed to call prepareRedeem() on chain: err=%v", err)
		return nil, nil, err
	}

	logger.WithField("prepareTx", common.ByteSliceToPureHexStr(tx_id)).Info("Prepare tx sent")
	return tx_id, ledger_number, nil
}

func (ctm *ChainTxMgr) procedurePrepare(ctx context.Context) error {
	// 0. Aquire necessary locks
	// 1. Find unprepared redeems from state db
	// 2. Filter those already prepared (tracked in mgr db)
	// 3. for each redeem, Check if the redeem is already prepared on chain
	// 4. for each redeem, Prepare redeem params (this step contains the schnorr signature request)
	// 5. for each redeem, Call prepareRedeem() function on chain
	// 6. for each redeem, Set Tx to the mgrdb, and it is "pending" status

	// 0. Aquire necessary locks
	ctm.mgrdbLock.Lock()
	defer ctm.mgrdbLock.Unlock()

	ctm.redeemLock.Lock()
	defer ctm.redeemLock.Unlock()

	// 1. Find unprepared redeems from state db
	redeems, err := ctm.QueryUnPrepared()
	if err != nil {
		logger.Errorf("failed to query unprepared redeems: err=%v", err)
		return err
	}

	// 2. Filter those already prepared (tracked in mgr db)
	redeems_clean, err := ctm.FilterUnPrepared(redeems)
	if err != nil {
		logger.Errorf("failed to filter unprepared redeems against mgrdb: err=%v", err)
		return err
	}

	if len(redeems_clean) == 0 {
		logger.Debug("no new redeems to process")
		return nil
	}

	for _, redeem := range redeems_clean {
		// 3. Check if the redeem is already prepared on chain
		found, err := ctm.IsDoublePrepare(redeem)
		if err != nil {
			logger.Errorf("failed to check if prepared: err=%v", err)
			continue
		}
		if found {
			continue
		}

		// 4. Prepare redeem params (this step contains the schnorr signature request)
		pp, err := ctm.PreparePrepare(ctx, redeem)
		if err != nil {
			logger.Errorf("failed to prepare redeem params: err=%v", err)
			continue
		}

		tx_id, ledger_number, err := ctm.CallPrepare(pp)
		if err != nil {
			logger.Errorf("failed to call prepareRedeem() on chain: err=%v", err)
			continue
		}

		// Extra: if the ledger_number is nil, we try the <best effort> to set it
		if ledger_number == nil {
			latest_ledger_number, err := ctm.chainWorker.GetLatestLedgerNumber()
			if err != nil {
				logger.Errorf("failed to get latest ledger number: err=%v", err)
			} else {
				if latest_ledger_number != nil {
					ledger_number = latest_ledger_number
				} else {
					logger.Errorf("latest ledger number is nil")
				}
			}
		}

		// 6. Set Tx to the mgrdb, and it is "pending" status
		err = ctm.SetPrepareToBeMonitored(pp, tx_id, ledger_number, agreement.Pending)
		if err != nil {
			logger.Errorf("failed to set mint to be monitored in mgrdb: err=%v", err)
			continue
		}
	}
	return nil
}

// This procedure tracks the Tx status on chain
// Then mark accordingly.
// If a Tx takes too long to be included, it will also marking it as timeout status.
// But it doesn't deal with timeout, it is another story.
func (ctm *ChainTxMgr) procedureMarkTxStatus() error {
	// 0. Aquire necessary locks
	// 1. Get all pending txs from mgr db
	// 2. For each pending tx, check the tx status on chain
	// 3. Update the tx status in mgr db
	// 4. If the tx takes too long to be accepted to blockchain, we consider it is timeout.

	// 0. Aquire necessary locks
	ctm.mgrdbLock.Lock()
	defer ctm.mgrdbLock.Unlock()

	// 1. Get all limbo/pending txs from mgr db
	pending_statuses := []agreement.MonitoredTxStatus{agreement.Limbo, agreement.Pending}
	pendingTxs, err := ctm.mgrdb.GetMonitoredTxByStatus(pending_statuses)
	if err != nil {
		logger.Errorf("failed to get monitored tx by status: err=%v", err)
		return err
	}

	if len(pendingTxs) == 0 {
		logger.Debug("no pending txs to process")
		return nil
	}

	for _, pendingTx := range pendingTxs {
		txId := pendingTx.TxIdentifier

		// 2. For each pending tx, check the tx status on chain
		status, err := ctm.chainWorker.GetTxStatus(txId)
		if err != nil {
			logger.Errorf("failed to get tx status on chain: err=%v", err)
			continue
		}

		pendingTx.TxStatus = status

		// 3. Update the tx status in mgr db
		err = ctm.mgrdb.UpdateTxStatus(txId, status)
		if err != nil {
			logger.Errorf("failed to update tx status in mgr db: err=%v", err)
			continue
		}

		// 4. If the tx takes too long to be accepted to blockchain, we consider it is timeout.
		included_statuses := []agreement.MonitoredTxStatus{agreement.Success, agreement.Reverted}
		if !agreement.UtilContains(included_statuses, status) {
			latestLedgerNumber, err := ctm.chainWorker.GetLatestLedgerNumber()
			if err != nil {
				logger.Errorf("failed to get latest ledger number: err=%v", err)
				continue
			}

			if latestLedgerNumber != nil && pendingTx.SentBlockchainLedgerNumber != nil {
				expireThreshold := new(big.Int).Add(pendingTx.SentBlockchainLedgerNumber, ctm.cfg.TimeoutTxLedgerNumber)
				if expireThreshold.Cmp(latestLedgerNumber) <= 0 { // eg. expireThreshold = 100; latestLedgerNumber = 120
					pendingTx.TxStatus = agreement.Timeout
					err = ctm.mgrdb.UpdateTxStatus(txId, agreement.Timeout)
					if err != nil {
						logger.Errorf("failed to update tx status to timeout in mgr db: err=%v", err)
						continue
					}
					logger.WithField("txId", common.ByteSliceToPureHexStr(txId)).Info("Tx marked as timeout")
				}
			}
		}

	}
	return nil
}

// Wait for Schnorr Signature.
// Then verify Signature.
func (ctm *ChainTxMgr) waitAndVerifySignature(ctx context.Context, msgHash [32]byte, ch <-chan *agreement.SignatureRequest) (*agreement.SignatureRequest, error) {
	newCtx, cancel := context.WithTimeout(ctx, ctm.cfg.TimeoutOnWaitingForSignature)
	defer cancel()

	for {
		select {
		case <-newCtx.Done():
			return nil, ctx.Err()
		case req := <-ch:
			if ok := common.Verify(ctm.pubKey[:], msgHash[:], req.Rx, req.S); !ok {
				return req, fmt.Errorf("ERR_BAD_SIGNATURE: signature verification failed")
			}
			return req, nil
		}
	}
}

// Query & Wait for BTC UTXO outpoints.
// If not enough outpoints returned, there can be error.
func (ctm *ChainTxMgr) waitForOutPoints(ctx context.Context, ch <-chan []agreement.BtcOutpoint) ([]agreement.BtcOutpoint, error) {
	newCtx, cancel := context.WithTimeout(ctx, ctm.cfg.TimeoutOnWaitingForOutpoints)
	defer cancel()

	for {
		select {
		case <-newCtx.Done():
			return nil, newCtx.Err()
		case outpoints := <-ch:
			if len(outpoints) == 0 {
				return nil, fmt.Errorf("ERR_EMPTY_OUTPOINTS: no outpoints returned")
			}
			return outpoints, nil
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
	// If ledger number field is really unknown, set to nil.
	// Return the (mint_tx_hash, sent_at_ledger_number, error)
	DoMint(mint *agreement.MintParameter) ([]byte, *big.Int, error)

	// Call the smart contract and verify if the redeem is already prepared on chain
	// The query uses the redeem's request tx id (prevent double prepare check)
	IsPrepared(requestTxId [32]byte) (bool, error)

	// Call the actual prepare() on smart contract on chain
	// Note: this function shall return the approximate ledger number when this tx is submitted to blockchain.
	// If ledger number field is really unknown, set to nil.
	// Return the (prepare_tx_hash, sent_at_ledger_number, error)
	DoPrepare(prepare *agreement.PrepareParameter) ([]byte, *big.Int, error)

	// Check Tx Status on Chain
	// Each transaction is to commit a change to blockchain,
	// Naturally, the status of the transaction can be 'success' or 'reverted'
	GetTxStatus(txId []byte) (agreement.MonitoredTxStatus, error)
}
