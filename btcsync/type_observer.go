package btcsync

import (
	"github.com/TEENet-io/bridge-go/btcaction"
)

/*
Observers got notified once an interested action is found
*/

type DepositObserver interface {
	GetNotifiedDeposit(da btcaction.DepositAction)
}

type WithdrawObserver interface {
	GetNotifiedWithdraw(wa btcaction.WithdrawAction)
}

type UnknownTransferObserver interface {
	GetNotifiedUnknownTransfer(uta btcaction.UnknownTransferAction)
}
