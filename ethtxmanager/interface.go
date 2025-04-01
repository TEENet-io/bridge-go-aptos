package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// This interface defines how Tx manager interact with BTC side wallet.
// What is the expected behavior of the btc wallet from btc side
type BtcWallet interface {
	// Request sends a request to the wallet to get outpoints for preparing
	// the redeem indexed by the tx hash and then return the outpoints via
	// the provided channel. The btc wallet should temporarily lock the
	// outpoints with a timeout. It should also monitor the RedeemPrepared
	// events emitted from the bridge for permanent locking.
	Request(
		reqTxId ethcommon.Hash, // eth requestTxHash
		amount *big.Int,
		ch chan<- []state.BtcOutpoint, // this channel receives a slice of outputs.
	) error
}
