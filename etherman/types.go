package etherman

import (
	"math/big"

	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type MintParams struct {
	BtcTxId  [32]byte
	Amount   *big.Int
	Receiver common.Address
	Rx       *big.Int
	S        *big.Int
}

type RequestParams struct {
	Auth     *bind.TransactOpts
	Amount   *big.Int
	Receiver string
}

type PrepareParams struct {
	RedeemRequestTxHash [32]byte
	Requester           common.Address
	Receiver            string
	Amount              *big.Int
	OutpointTxIds       [][32]byte
	OutpointIdxs        []uint16
	Rx                  *big.Int
	S                   *big.Int
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
