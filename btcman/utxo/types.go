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
	TxID      string           // Tx Identifier, human readable hex string
	TxHash    *chainhash.Hash  // Tx Identifier, used for tx search in bitcoin nodes
	Vout      uint32           // at which index of the Tx's outputs is the UTXO
	Amount    int64            // in satoshi
	PkScriptT PubKeyScriptType // Type of the locking script
	PkScript  []byte           // Locking Script itself
}

// Return a human-readable amount in BTC
// eg. 1e8 (satoshi) = 1.0 (BTC)
func (u *UTXO) AmountHuman() float64 {
	return float64(u.Amount) / 1e8
}
