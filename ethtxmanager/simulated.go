package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type MockBtcWallet struct {
}

func (w *MockBtcWallet) Request(
	Id ethcommon.Hash,
	amount *big.Int,
	ch chan<- []state.BtcOutpoint,
) error {
	outpoints := []state.BtcOutpoint{
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
		TxHash:       common.RandBytes32(),
		Id:           common.RandBytes32(),
		SentAfter:    common.RandBytes32(),
		SentAfterBlk: 0,
		MinedAt:      minedAt,
		Status:       status,
	}

	return mt
}
