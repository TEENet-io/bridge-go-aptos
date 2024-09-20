package etherman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type MintParams struct {
	BtcTxId  [32]byte
	Amount   *big.Int
	Receiver ethcommon.Address
	Rx       *big.Int
	S        *big.Int
}

type RequestParams struct {
	Auth     *bind.TransactOpts
	Amount   *big.Int
	Receiver string
}

type PrepareParams struct {
	RequestTxHash [32]byte
	Requester     ethcommon.Address
	Receiver      string
	Amount        *big.Int
	OutpointTxIds [][32]byte
	OutpointIdxs  []uint16
	Rx            *big.Int
	S             *big.Int
}

func (p *PrepareParams) SigningHash() [32]byte {
	outpointIdxs := []*big.Int{}
	for _, idx := range p.OutpointIdxs {
		outpointIdxs = append(outpointIdxs, big.NewInt(int64(idx)))
	}

	return crypto.Keccak256Hash(common.EncodePacked(
		p.RequestTxHash,
		p.Requester.String(),
		string(p.Receiver),
		p.Amount,
		p.OutpointTxIds,
		outpointIdxs,
	))
}

type RedeemRequestedEvent struct {
	bridge.TEENetBtcBridgeRedeemRequested
	TxHash [32]byte
}

type RedeemPreparedEvent struct {
	bridge.TEENetBtcBridgeRedeemPrepared
	TxHash [32]byte
}

type MintedEvent struct {
	bridge.TEENetBtcBridgeMinted
	TxHash [32]byte
}
