package eth2btcstate

import (
	"errors"
	"fmt"
	"math/big"
)

type ModifiyStateError struct{}

func (e *ModifiyStateError) CannotPrepareDueToRequestedRedeemNotFound(txHash []byte) error {
	msg := fmt.Sprintf("cannot be prepared due to non-existing requested redeem and : txhash=0x%x", txHash)
	return errors.New(msg)
}

func (e *ModifiyStateError) CannotPrepareDueToRequestedRedeemInvalid(txHash []byte) error {
	msg := fmt.Sprintf("cannot be prepared due to invalid requested redeem: txHash=0x%x", txHash)
	return errors.New(msg)
}

func (e *ModifiyStateError) CannotPrepareDueToInvalidStatus(status RedeemStatus) error {
	msg := fmt.Sprintf("cannot be prepared due to invalid status: status=%s", status)
	return errors.New(msg)
}

func (e *ModifiyStateError) StoredFinalizedBlockNumberLessThanStartingBlockNumber(num *big.Int) error {
	msg := fmt.Sprintf("stored finalized block number less than the starting block number: fbNum=%v", num)
	return errors.New(msg)
}

type ModifyStateWarning struct{}

func (w *ModifyStateWarning) RedeemAlreadyExists(txHash []byte) string {
	return fmt.Sprintf("redeem already exists: txHash=0x%x", txHash)
}

func (w *ModifyStateWarning) RedeemAlreadyPreparedOrCompleted(txHash []byte) string {
	return fmt.Sprintf("redeem already prepared or completed: txHash=0x%x", txHash)
}

func (w *ModifyStateWarning) NewFinalizedBlockNumberLessThanStored(newFinalized, stored *big.Int) string {
	return fmt.Sprintf("new finalized block number less than the stored one: new=%v, stored=%v", newFinalized, stored)
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
