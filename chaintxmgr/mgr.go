package chaintxmgr

import (
	"context"
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/chaintxmgrdb"
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

// Mgr's worker on chain, do the dirty job.
type MgrWorker interface {
}
