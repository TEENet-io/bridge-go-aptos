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
func (params *MintParameter) GenerateMsgHash() common.Hash {
	return crypto.Keccak256Hash(mycommon.EncodePacked(
		params.BtcTxId,
		params.Receiver,
		params.Amount,
	))
}

// To Prepare a redeem on chain (eth/aptos),
// Following params are needed to provide info.
// Real params to call Ethereum contract Redeem's Prepare()
type PrepareParameter struct {
	RequestTxHash common.Hash // [32]byte
	Requester     []byte      // [20]byte = eth address, [32]byte = aptos address
	Receiver      string      // btc address, cannot be represented in bytes...
	Amount        *big.Int
	OutpointTxIds []common.Hash // each is [32]byte
	OutpointIdxs  []uint16      // corresponding output's vout to btc_tx_id(s)
	Rx            *big.Int
	S             *big.Int
}

// Serialize the parameters and create a hash
func (p *PrepareParameter) GenerateMsgHash() common.Hash {
	outpointIdxs := []*big.Int{}
	for _, idx := range p.OutpointIdxs {
		outpointIdxs = append(outpointIdxs, big.NewInt(int64(idx)))
	}

	return crypto.Keccak256Hash(mycommon.EncodePacked(
		p.RequestTxHash,
		p.Requester,
		string(p.Receiver),
		p.Amount,
		p.OutpointTxIds,
		outpointIdxs,
	))
}

// Create RedeemPrepare Tx needed params from state.redeem object.
// This is a Helper function for Redeem's prepare.
// TODO: Unify the BTC UTXO's representation of TXId + Vout
func ConvertOutpoints(bop []BtcOutpoint) ([]common.Hash, []uint16) {
	outpointTxIds := []common.Hash{}
	outpointIdxs := []uint16{}

	for _, outpoint := range bop {
		outpointTxIds = append(outpointTxIds, outpoint.BtcTxId)
		outpointIdxs = append(outpointIdxs, outpoint.BtcIdx)
	}

	return outpointTxIds, outpointIdxs
}

// Enum for the status of the tx submitted to the blockchain.
type MonitoredTxStatus string

const (
	MalForm  MonitoredTxStatus = "malform"  // mal-form Tx, cannot be accepted by blockchain.
	Limbo    MonitoredTxStatus = "limbo"    // Tx sent, but not found anywhere.
	Pending  MonitoredTxStatus = "pending"  // pending in the blockchain's mempool, not executed, yet.
	Success  MonitoredTxStatus = "success"  // Tx success, included in the blockchain ledger.
	Reverted MonitoredTxStatus = "reverted" // Tx execution failed, wanted change doesn't apply to blockchain.
	Timeout  MonitoredTxStatus = "timeout"  // For too long a time, it is not success or reverted.
	Reorg    MonitoredTxStatus = "reorg"    // blockchain re-orged (rarely used, not used in program yet)
)
