package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/state/eth2btcstate"
)

type SchnorrThresholdWallet interface {
	// Sign sends a request to the wallet to sign on the signing hash
	// and return the signature via the provided channel
	Sign(request *SignatureRequest, ch chan<- *SignatureRequest) error
}

type BtcWallet interface {
	// Request sends a request to the wallet to get outpoints for preparing
	// the redeem indexed by the tx hash and then return the outpoints via
	// the provided channel. The btc wallet should temporarily lock the
	// outpoints with a timeout. It should also monitor the RedeemPrepared
	// events emitted from the bridge for permanent locking.
	Request(
		requestRedeemTxHash [32]byte,
		amount *big.Int,
		ch chan<- []eth2btcstate.Outpoint,
	) error
}
