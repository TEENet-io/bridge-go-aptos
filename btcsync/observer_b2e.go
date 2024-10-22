package btcsync

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// This file implements an observer
// that writes "mint" table in shared state.
// (which later will trigger BTC2EVM Mint on ETH side).

type BTC2EVMObserver struct {
	sharedState *state.State
	Ch          chan btcaction.DepositAction // communication channel
}

func NewBTC2EVMObserver(state *state.State, bufferSize int) *BTC2EVMObserver {
	return &BTC2EVMObserver{
		sharedState: state,
		Ch:          make(chan btcaction.DepositAction, bufferSize),
	}
}

// GetNotifiedDeposit implements the DepositObserver interface
// it writes to shared state to allow following actions being taken.
// You should call it as a separate goroutine (with go)
func (s *BTC2EVMObserver) GetNotifiedDeposit() {
	for data := range s.Ch {
		// type conversion
		m := state.Mint{
			BtcTxId:    ethcommon.HexToHash(data.Basic.TxHash),
			MintTxHash: common.EmptyHash, // this field is empty until ETH side mints TWBTC.
			Receiver:   ethcommon.HexToAddress(data.EvmAddr),
			Amount:     new(big.Int).SetInt64(data.DepositValue),
		}

		// write to state directly.
		s.sharedState.SetNewBTC2EVMMint(&m)
	}
}
