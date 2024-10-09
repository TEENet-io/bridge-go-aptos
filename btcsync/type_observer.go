package btcsync

/*
Observers got notified once an interested action is found
*/

type DepositObserver interface {
	GetNotifiedDeposit()
}

type WithdrawObserver interface {
	GetNotifiedWithdraw()
}

type UnknownTransferObserver interface {
	GetNotifiedUnknownTransfer()
}

type UTXOObserver interface {
	GetNotifiedUtxo()
}
