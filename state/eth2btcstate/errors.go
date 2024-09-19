package eth2btcstate

import (
	"errors"
	"fmt"
	"math/big"
)

type StateError struct{}

func (e *StateError) CannotPrepareDueToRedeemRequestInvalid(txHash []byte) error {
	msg := fmt.Sprintf("cannot be prepared since the requested redeem is invalid: txHash=0x%x", txHash)
	return errors.New(msg)
}

func (e *StateError) StoredFinalizedBlockNumberLessThanStartingBlockNumber(num *big.Int) error {
	msg := fmt.Sprintf("stored finalized block number less than the starting block number: fbNum=%v", num)
	return errors.New(msg)
}

type StateDBError struct{}

func (e *StateDBError) CannotUpdateDueToInvalidStatus(redeem *Redeem) error {
	msg := fmt.Sprintf("cannot update due to invalid status (expect status == prepared): redeem=%v", redeem)
	return errors.New(msg)
}

func (e *StateDBError) CannotInsertDueToInvalidStatus(redeem *Redeem) error {
	msg := fmt.Sprintf("cannot insert due to invalid status (expect status == requested | invalid): redeem=%v", redeem)
	return errors.New(msg)
}
