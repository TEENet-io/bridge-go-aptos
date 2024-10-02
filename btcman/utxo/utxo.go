/*
This file contains filter/select operations on UTXO.
*/
package utxo

import (
	"errors"
)

// Filter UTXO according to PubKeyScript (locking script) type
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

// Choose some UTXO(s) for future spending.
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
