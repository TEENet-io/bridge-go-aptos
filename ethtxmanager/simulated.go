package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
)

type MockBtcWallet struct{}

func (w *MockBtcWallet) Request(
	reqId []byte,
	amount *big.Int,
	ch chan<- []agreement.BtcOutpoint,
) error {
	outpoints := []agreement.BtcOutpoint{
		{
			BtcTxId: common.RandBytes32(),
			BtcIdx:  0,
		},
	}

	ch <- outpoints

	return nil
}

func RandMonitoredTx(status MonitoredTxStatus, outpointNum int) *MonitoredTx {
	minedAt := common.EmptyHash

	if status != Pending {
		minedAt = common.RandBytes32()
	}

	mt := &MonitoredTx{
		TxHash:        common.RandBytes32(),
		RefIdentifier: common.RandBytes32(),
		SentAfter:     common.RandBytes32(),
		SentAfterBlk:  0,
		MinedAt:       minedAt,
		Status:        status,
	}

	return mt
}
