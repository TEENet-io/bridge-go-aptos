/*
 */
package btcsync

import (
	"github.com/TEENet-io/bridge-go/btcvault"
)

type ObservedUTXO struct {
	BlockNumber int32
	BlockHash   string
	TxID        string
	Vout        int32
	Amount      int64
}

type ObserverUTXOVault struct {
	Backend *btcvault.TreasureVault
	Ch      chan ObservedUTXO
}

func NewObserverUTXOVault(backend *btcvault.TreasureVault, bufferSize int) *ObserverUTXOVault {
	return &ObserverUTXOVault{
		Backend: backend,
		Ch:      make(chan ObservedUTXO, bufferSize),
	}
}

func (o *ObserverUTXOVault) GetNotifiedUtxo() {
	for data := range o.Ch {
		o.Backend.AddUTXO(data.BlockNumber, data.BlockHash, data.TxID, data.Vout, data.Amount)
	}
}
