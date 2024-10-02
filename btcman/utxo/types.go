/*
This file contains low-level custom data structures used accross the program related to bitcoin.
  - PubKeyScriptType: the locking script type (as part of UTXO)
  - UTXO, the unspend transaction output.
*/
package utxo

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// PubKeyScript (LockingScript) type
type PubKeyScriptType int

// Enumerate of PubKeyScriptType
const (
	ANY_SCRIPT_T = iota
	P2PKH_SCRIPT_T
	P2WPKH_SCRIPT_T
)

// Represents the unspent transaction output (UTXO)
// in our program
type UTXO struct {
	TxID      string           // Identifier, human readable
	TxHash    *chainhash.Hash  // Identifier, used for tx search
	Vout      uint32           // exact index of the Tx's outputs to be spent
	Amount    int64            // in satoshi
	PkScriptT PubKeyScriptType // Type of the locking script
	PkScript  []byte           // Locking Script itself
}

// Return a human-readable amount in BTC
// eg. 1e8 (satoshi) = 1.0 (BTC)
func (u *UTXO) AmountHuman() float64 {
	return float64(u.Amount) / 1e8
}
