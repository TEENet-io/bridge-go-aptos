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
