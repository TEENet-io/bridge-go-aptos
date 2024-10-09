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

type ObserverBTCVault struct {
	Backend *btcvault.TreasureVault
	Ch      chan ObservedUTXO
}

func NewObserverBTCVault(backend *btcvault.TreasureVault, bufferSize int) *ObserverBTCVault {
	return &ObserverBTCVault{
		Backend: backend,
		Ch:      make(chan ObservedUTXO, bufferSize),
	}
}

func (o *ObserverBTCVault) GetNotifiedUtxo() {
	for data := range o.Ch {
		o.Backend.AddUTXO(data.BlockNumber, data.BlockHash, data.TxID, data.Vout, data.Amount)
	}
}
