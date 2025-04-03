package etherman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Real params to call Ethereum contract mint()
type MintParams struct {
	BtcTxId  ethcommon.Hash // bitcoin transaction hash
	Amount   *big.Int
	Receiver []byte   // ethereum address
	Rx       *big.Int // part of schnorr signature
	S        *big.Int // part of schnorr signature
}

func (params *MintParams) SigningHash() ethcommon.Hash {
	return crypto.Keccak256Hash(common.EncodePacked(
		params.BtcTxId,
		params.Receiver,
		params.Amount,
	))
}

// Real params to call Ethereum contract Redeem's Request()
type RequestParams struct {
	Amount   *big.Int
	Receiver string
}

// Real params to call Ethereum contract Redeem's Prepare()
type PrepareParams struct {
	RequestTxHash ethcommon.Hash // [32]byte
	Requester     []byte         // [20]byte = eth address, [32]byte = aptos address
	Receiver      string         // btc address, cannot be represented in bytes...
	Amount        *big.Int
	OutpointTxIds []ethcommon.Hash // [32]byte = btc_tx_id
	OutpointIdxs  []uint16         // number, corresponding output vout to btc_tx_id(s)
	Rx            *big.Int
	S             *big.Int
}

// seraialize the parameters and create a hash
func (p *PrepareParams) SigningHash() ethcommon.Hash {
	outpointIdxs := []*big.Int{}
	for _, idx := range p.OutpointIdxs {
		outpointIdxs = append(outpointIdxs, big.NewInt(int64(idx)))
	}

	return crypto.Keccak256Hash(common.EncodePacked(
		p.RequestTxHash,
		p.Requester,
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
