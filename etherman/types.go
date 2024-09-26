package etherman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type MintParams struct {
	BtcTxId  ethcommon.Hash
	Amount   *big.Int
	Receiver ethcommon.Address
	Rx       *big.Int
	S        *big.Int
}

func (params *MintParams) SigningHash() ethcommon.Hash {
	return crypto.Keccak256Hash(common.EncodePacked(
		params.BtcTxId,
		params.Amount,
		params.Receiver,
	))
}

type RequestParams struct {
	Amount   *big.Int
	Receiver string
}

type PrepareParams struct {
	RequestTxHash ethcommon.Hash
	Requester     ethcommon.Address
	Receiver      string
	Amount        *big.Int
	OutpointTxIds []ethcommon.Hash
	OutpointIdxs  []uint16
	Rx            *big.Int
	S             *big.Int
}

func (p *PrepareParams) SigningHash() ethcommon.Hash {
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
