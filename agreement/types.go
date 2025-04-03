// Golbal Agreement on types

package agreement

import (
	"fmt"
	"math/big"

	mycommon "github.com/TEENet-io/bridge-go/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// MintedEvent reqpresents when TWBTC is minted
// on ETH side.
type MintedEvent struct {
	MintTxHash common.Hash
	BtcTxId    common.Hash
	Receiver   []byte
	Amount     *big.Int
}

func (ev *MintedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}

// RedeemRequestedEvent is the event when user
// requests a redeem (EVM2BTC).
type RedeemRequestedEvent struct {
	RequestTxHash   common.Hash
	Requester       []byte // [20]byte = ethereum address, [32]byte = aptos address
	Receiver        string
	Amount          *big.Int
	IsValidReceiver bool
}

// Debug
func (ev *RedeemRequestedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}

// RedeemPreparedEvent is the event when
// we have prepared the BTC redeem (BTC2EVM).
// The UTXO(s) are chosen.
// But it hasn't been sent out on BTC side yet.
type RedeemPreparedEvent struct {
	RequestTxHash common.Hash
	PrepareTxHash common.Hash
	Requester     []byte // [20]byte = ethereum address, [32]byte = aptos address
	Receiver      string
	Amount        *big.Int
	OutpointTxIds []common.Hash
	OutpointIdxs  []uint16
}

func (ev *RedeemPreparedEvent) String() string {
	return fmt.Sprintf("%+v", *ev)
}

type BtcOutpoint struct {
	BtcTxId common.Hash
	BtcIdx  uint16
}

type JSONBtcOutpoint struct {
	BtcTxId string `json:"txid"`
	BtcIdx  uint16 `json:"idx"`
}

// This is the type that Tx manager will expect
// to request the signature from signature provider. (whether local or remote)
type SignatureRequest struct {
	Id          common.Hash
	SigningHash common.Hash
	Outpoints   []BtcOutpoint
	Rx          *big.Int
	S           *big.Int
}

// To mint on chain (eth/aptos),
// Following params are needed to provide info.
type MintParameter struct {
	BtcTxId  common.Hash // bitcoin transaction hash is always 32 byte
	Amount   *big.Int
	Receiver []byte   // ethereum address
	Rx       *big.Int // part of schnorr signature
	S        *big.Int // part of schnorr signature
}

// The msg-hash for sign
func (params *MintParameter) GenerateSigningHash() common.Hash {
	return crypto.Keccak256Hash(mycommon.EncodePacked(
		params.BtcTxId,
		params.Receiver,
		params.Amount,
	))
}
