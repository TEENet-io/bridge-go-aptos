package ethsync

import (
	"errors"
	"fmt"
	"math/big"
)

func ErrChainIDUnmatched(expected, actual *big.Int) error {
	msg := fmt.Sprintf("chain ID mismatch: expected=%v, actual=%v", expected, actual)
	return errors.New(msg)
}

func ErrStoredFinalizedBlockNumberInvalid(stored, startingBlock *big.Int) error {
	msg := fmt.Sprintf("stored finalized block number is less than starting block number: stored=%v, starting=%v", stored, startingBlock)
	return errors.New(msg)
}
