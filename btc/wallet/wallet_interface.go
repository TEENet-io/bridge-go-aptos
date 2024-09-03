package wallet

import (
	"github.com/btcsuite/btcd/wire"
	"teenet.io/bridge-go/btc/data"
)

// Locking defines the actions
// that produce the locking part of a Tx (the outputs).
// Each action adds an output clause to the outputs of a Tx.
type Locking interface {
	// Pay to P2PKH type of address only
	AppendOutputP2PKH(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error)
	// Pay to P2WPKH type of address only
	AppendOutputP2WPKH(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error)
	// Pay to any type of address (not script address, though)
	AppendOutputPayToAddress(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error)
}

// Unlocking defins the actions
// that produce the unlocking part of a Tx (the inputs).
type Unlocking interface {
	// Given a list of UTXO(s), unlock each UTXO and add to section.
	// How to unlock? it depends on the implementation.
	// eg. single private key sign, multi-sig
	Unlock(tx *wire.MsgTx, prevOutputs []data.UTXO) (*wire.MsgTx, error)
}
