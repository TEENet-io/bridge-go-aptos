/*
This file contains filter/select operations on UTXO.
*/
package utxo

import (
	"errors"
)

// Choose some UTXO(s) for future spending.
// Collect several UTXO, the sum to be larger than (amount + fee).
// Error if cannot collect enough satisfy the requriement.
func SelectUtxo(inputs []*UTXO, amount int64, fee int64) ([]*UTXO, error) {
	var sum int64
	topIdx := 0
	flag := false
	for idx, item := range inputs {
		sum += item.Amount
		topIdx = idx
		if sum > (amount + fee) {
			flag = true
			break
		}
	}
	if !flag {
		return nil, errors.New("cannot satisfy requirement")
	} else {
		return inputs[:topIdx+1], nil
	}
}
