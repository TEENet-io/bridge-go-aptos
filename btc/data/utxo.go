package data

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Locking script type
type PubKeyScriptType int

// Enumerate of PubKeyScriptType
const (
	ANY_SCRIPT_T = iota
	P2PKH_SCRIPT_T
	P2WPKH_SCRIPT_T
)

// UTXO represents the unspent transaction output
type UTXO struct {
	TxID      string           // Identifier, human readable
	TxHash    *chainhash.Hash  // Identifier, used for tx search
	Vout      uint32           // exact index of the Tx's outputs to be spent
	Amount    int64            // in satoshi
	PkScriptT PubKeyScriptType // Type of the locking script
	PkScript  []byte           // Locking Script itself
}

// Return a human readable amount in BTC
// eg. Amount = 1e8 (satoshi) = 1.0 (BTC)
func (u *UTXO) AmountHuman() float64 {
	return float64(u.Amount) / 1e8
}

// Filter UTXO according to locking script type
func FilterUtxo(inputs []UTXO, wanted PubKeyScriptType) []UTXO {
	var r []UTXO
	for _, item := range inputs {
		if wanted == ANY_SCRIPT_T {
			r = append(r, item)
		} else {
			if item.PkScriptT == wanted {
				r = append(r, item)
			}
		}
	}
	return r
}

// Choose some UTXO(s) for future combination.
// Collect several UTXO, the sum to be larger than (amount + fee).
// Error if cannot collect enough satisfy the requriement.
func SelectUtxo(inputs []UTXO, amount int64, fee int64) ([]UTXO, error) {
	var sum int64
	top_idx := 0
	flag := false
	for idx, item := range inputs {
		sum += item.Amount
		top_idx = idx
		if sum > (amount + fee) {
			flag = true
			break
		}
	}
	if !flag {
		return nil, errors.New("cannot satisfy requirement")
	} else {
		return inputs[:top_idx+1], nil
	}
}
