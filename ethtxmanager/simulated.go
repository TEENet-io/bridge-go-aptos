package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/state"
)

type MockBtcWallet struct {
}

func (w *MockBtcWallet) Request(
	requestRedeemTxHash [32]byte,
	amount *big.Int,
	ch chan<- []state.Outpoint,
) error {
	outpoints := []state.Outpoint{
		{
			TxId: common.RandBytes32(),
			Idx:  0,
		},
	}

	ch <- outpoints

	return nil
}

type MockSchnorrThresholdWallet struct {
	sim *etherman.SimEtherman
}

func (w *MockSchnorrThresholdWallet) Sign(
	request *SignatureRequest,
	ch chan<- *SignatureRequest,
) error {
	rx, s, err := w.sim.Sign(request.SigningHash[:])

	// if failed to sign, do nothing to allow the routine
	// that waits for the signature to timeout
	if err != nil {
		return err
	}

	ch <- &SignatureRequest{
		RequestTxHash: request.RequestTxHash,
		SigningHash:   request.SigningHash,
		Rx:            rx,
		S:             s,
	}

	return nil
}

func RandMonitoredTx(status MonitoredTxStatus, outpointNum int) *MonitoredTx {
	minedAt := common.EmptyHash

	if status != Pending {
		minedAt = common.RandBytes32()
	}

	mt := &MonitoredTx{
		TxHash:      common.RandBytes32(),
		Id:          common.RandBytes32(),
		SigningHash: common.RandBytes32(),
		SentAfter:   common.RandBytes32(),
		MinedAt:     minedAt,
		Status:      status,
	}

	for i := 0; i < outpointNum; i++ {
		mt.Outpoints = append(mt.Outpoints, state.Outpoint{
			TxId: common.RandBytes32(),
			Idx:  uint16(i),
		})
	}

	rx := common.RandBytes32()
	mt.Rx = new(big.Int).SetBytes(rx[:])
	s := common.RandBytes32()
	mt.S = new(big.Int).SetBytes(s[:])

	return mt
}
