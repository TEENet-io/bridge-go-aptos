/*
Locker and Unlocker are the basic interfaces
that a tx assembler shall satisfy.

By imlementing Locker, the tx assembler
can add pay output to P2PKH/P2WPKH receiver.

By implementing Unlocker, the tx assembler
can unlock UTXOs (inputs) previously received.

Remember:
Always create the "lock" part firstly on Tx, then create the "unlock" part on Tx.
Otherwise the Tx verfication may fail.
*/
package assembler

import (
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

// Locker defines the actions
// that produce the "locking" part of a Tx.
// Each action below adds an output clause to the outputs of a Tx.
type Locker interface {
	// Add a P2PKH (pay-to-public-key-hash) clause to Tx.
	// This means adding a legacy btc address receiver as fund receiver.
	// amount is in satoshi.
	AddP2PKH(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error)
	// Add a P2WPKH (pay-to-witness-public-key-hash) clause to Tx.
	// This means adding a segwit btc address receiver as fund receiver.
	// amount is in satoshi.
	AddP2WPKH(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error)
	// Add a pay-to-any-type-of-address clause to Tx.
	// (cannot be type of script address, though)
	AddPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error)
}

// Unlocker defins the actions
// that produce the "unlocking" part of a Tx (aka the inputs).
// Call Unlock() on a list of UTXO to unlock them (produce valid signature to spend each UTXO)
type Unlocker interface {
	// Given a list of UTXO(s), unlock each UTXO and add to unlocking section of MsgTx.
	// How to unlock? it depends on the specific wallet implementation.
	// eg. single private key signature, multi-sig, m-n schnorr sign, etc.
	Unlock(tx *wire.MsgTx, prevOutputs []utxo.UTXO) (*wire.MsgTx, error)
}
