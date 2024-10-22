package btcsync

import (
	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type RedeemObserver struct {
	sharedState *state.State
	Ch          chan btcaction.RedeemAction // communication channel
}

func NewRedeemObserver(state *state.State, bufferSize int) *RedeemObserver {
	return &RedeemObserver{
		sharedState: state,
		Ch:          make(chan btcaction.RedeemAction, bufferSize),
	}
}

// Notify that the redeem on BTC is totally completed.
func (r *RedeemObserver) GetNotifiedRedeemCompleted() {
	for data := range r.Ch {
		r.sharedState.SetRedeemCompleted(
			ethcommon.HexToHash(data.EthRequestTxID),
			ethcommon.HexToHash(data.BtcHash),
		)
	}
}
