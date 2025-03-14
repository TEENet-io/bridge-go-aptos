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

type Operator interface {
	// BTC Locking script producer
	// This action usually <DOES NOT> require any private key or sign process.
	// that produces the "locking" part of a Tx.
	// "locking" adds an output clause to the outputs of a BTC Tx.

	// Add a pay-to-any-type-of-address clause to Tx.
	// (the address cannot be type of script address, though)
	AppendPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error)

	// BTC Unlocking script producer
	// that produces the "unlocking" part of a Tx (aka the inputs).
	// This action <REQUIRE> a private key or sign process.

	// Call Unlock() on a list of UTXO to unlock them (produce valid signature to spend each UTXO)
	// Given a list of UTXO(s), unlock each UTXO and add to unlocking section of MsgTx.
	// How to unlock? it depends on the specific wallet implementation.
	// eg. single private key signature, muzlti-sig, m-n schnorr sign, etc.
	Unlock(tx *wire.MsgTx, prevOutputs []*utxo.UTXO) (*wire.MsgTx, error)
}
