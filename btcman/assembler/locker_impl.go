package assembler

/*
This file implements BTC "Locker" interface.

Since locking scripts do not require any prior knowledge of private keys,
it is universal to all wallet implementations.

So we can do it here.
*/

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// Locking produces a script (to be solved by future spender)
// It doesn't require a private key/sign process.
func AppendPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	// Decode the receiver's address
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, dst_chain_cfg)
	if err != nil {
		return nil, err
	}

	// Create a pay-to-address script
	// (the lib handles different types of address, like P2PKH, P2WPKH, etc.)
	txOutScript, err := txscript.PayToAddrScript(btcDstAddress)
	if err != nil {
		return nil, err
	}

	// Attach the pay-to-address script to the Tx output
	txOut := wire.NewTxOut(amount, txOutScript)
	tx.AddTxOut(txOut)

	// return the modified tx
	return tx, nil
}
