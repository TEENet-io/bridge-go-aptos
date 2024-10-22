package btcsync

/*
Observers got notified once an interested action is found
*/

// Observer on deposit
type DepositObserver interface {
	GetNotifiedDeposit()
}

// Observer on redeem completed.
type RedeemCompletedObserver interface {
	GetNotifiedRedeemCompleted()
}

// Observer on other transfer than deposit
type OtherTransferObserver interface {
	GetNotifiedOtherTransfer()
}

// Observer on new UTXO
type UTXOObserver interface {
	GetNotifiedUtxo()
}
