package chaintxmgr

import (
	"time"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/chaintxmgrdb"
)

type ChainTxMgrConfig struct {
	IntervalCheckTime time.Duration
}

type ChainTxMgr struct {
	mgrdb            chaintxmgrdb.ChainTxMgrDB
	schnorrWallet    agreement.SchnorrAsyncSigner
	btcUTXOResponder agreement.BtcUTXOResponder
}
