package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type RedeemStatus string

const (
	RedeemStatusRequested RedeemStatus = "requested" // user requested
	RedeemStatusPrepared  RedeemStatus = "prepared"  // bridge prepared utxos for request, not sent on btc yet.
	RedeemStatusCompleted RedeemStatus = "completed" // bridge sent btc tx and btc tx is mined.
	RedeemStatusInvalid   RedeemStatus = "invalid"
)

var (
	ErrorAmountInvalid          = errors.New("amount invalid")
	ErrorRequesterInvalid       = errors.New("requester address invalid")
	ErrorRequestTxHashInvalid   = errors.New("redeem request tx hash invalid")
	ErrorRequestTxHashUnmatched = errors.New("redeem request tx hash unmatched")
	ErrorRequesterUnmatched     = errors.New("requester unmatched")
	ErrorAmountUnmatched        = errors.New("amount unmatched")
	ErrorReceiverUnmatched      = errors.New("receiver unmatched")
	ErrorOutpointsInvalid       = errors.New("outpoints invalid")
	ErrorOutpointTxIdInvalid    = errors.New("outpoint tx id invalid")
	ErrorRequireStatusRequested = errors.New("require status == requested")
	ErrorPrepareTxHashEmpty     = errors.New("prepare tx hash is empty")
)

type Redeem struct {
	RequestTxHash ethcommon.Hash // [32]byte
	PrepareTxHash ethcommon.Hash // [32]byte
	BtcTxId       ethcommon.Hash // [32]byte
	Requester     []byte         // [20]byte = ethereum address, [32]byte = aptos address
	Receiver      string         // receiver btc address, cannot be represented in bytes..
	Amount        *big.Int
	Outpoints     []agreement.BtcOutpoint
	Status        RedeemStatus // string
}

// Created a new Redeem object when user requests.
// redeem.status = invalid if the receiver is invalid.
// redeem.status = rquested if everything is okay.
func createRedeemFromRequestedEvent(ev *agreement.RedeemRequestedEvent) (*Redeem, error) {
	r := &Redeem{}

	if ev.RequestTxHash == [32]byte{} {
		return nil, ErrorRequestTxHashInvalid
	}

	// if ev.Requester == [20]byte{} {
	// 	return nil, ErrorRequesterInvalid
	// }

	if ev.Amount == nil || ev.Amount.Sign() <= 0 {
		return nil, ErrorAmountInvalid
	}

	r.RequestTxHash = ev.RequestTxHash
	r.Requester = ev.Requester
	r.Amount = new(big.Int).Set(ev.Amount)
	r.Receiver = ev.Receiver
	if ev.IsValidReceiver {
		r.Status = RedeemStatusRequested
	} else {
		r.Status = RedeemStatusInvalid
	}

	return r, nil
}

// This function is important.
// It updates the "Redeem" object from a "RedeemPrepared" event.
// The "Redeem" status is switched from "requested" to "prepared".
func (r *Redeem) updateFromPreparedEvent(ev *agreement.RedeemPreparedEvent) (*Redeem, error) {
	if ev.RequestTxHash != r.RequestTxHash {
		return nil, ErrorRequestTxHashUnmatched
	}

	if !common.CompareSlices(ev.Requester, r.Requester) {
		return nil, ErrorRequesterUnmatched
	}

	if ev.Amount.Cmp(r.Amount) != 0 {
		return nil, ErrorAmountUnmatched
	}

	if ev.Receiver != r.Receiver {
		return nil, ErrorReceiverUnmatched
	}

	if ev.PrepareTxHash == [32]byte{} {
		return nil, ErrorPrepareTxHashEmpty
	}

	// Check
	if len(ev.OutpointTxIds) == 0 ||
		len(ev.OutpointIdxs) == 0 ||
		len(ev.OutpointTxIds) != len(ev.OutpointIdxs) {
		return nil, ErrorOutpointsInvalid
	}

	// Check
	for i := range ev.OutpointIdxs {
		if ev.OutpointTxIds[i] == [32]byte{} {
			return nil, ErrorOutpointTxIdInvalid
		}
	}

	// Set Prepare ETH Transaction Hash
	r.PrepareTxHash = ev.PrepareTxHash

	// Set the status to be "prepared"
	// This is where the real "prepared" status is set to a redeem record.
	r.Status = RedeemStatusPrepared

	// Set the outpoints
	r.Outpoints = []agreement.BtcOutpoint{}
	for i := range ev.OutpointTxIds {
		r.Outpoints = append(r.Outpoints, agreement.BtcOutpoint{
			BtcTxId: ev.OutpointTxIds[i],
			BtcIdx:  ev.OutpointIdxs[i],
		})
	}

	return r, nil
}

func createRedeemFromPreparedEvent(ev *agreement.RedeemPreparedEvent) (*Redeem, error) {
	r, err := createRedeemFromRequestedEvent(&agreement.RedeemRequestedEvent{
		RequestTxHash:   ev.RequestTxHash,
		Requester:       ev.Requester,
		Receiver:        ev.Receiver,
		Amount:          new(big.Int).Set(ev.Amount),
		IsValidReceiver: true,
	})

	if err != nil {
		return nil, err
	}

	return r.updateFromPreparedEvent(ev)
}

func (r *Redeem) MarshalJSON() ([]byte, error) {
	jOutpoint := []agreement.JSONBtcOutpoint{}
	for _, outpoint := range r.Outpoints {
		jOutpoint = append(jOutpoint, agreement.JSONBtcOutpoint{
			BtcTxId: outpoint.BtcTxId.String(),
			BtcIdx:  outpoint.BtcIdx,
		})
	}

	return json.Marshal(&JSONRedeem{
		RequestTxHash: r.RequestTxHash.String(),
		PrepareTxHash: r.PrepareTxHash.String(),
		BtcTxId:       r.BtcTxId.String(),
		Requester:     common.Prepend0xPrefix(common.ByteSliceToPureHexStr(r.Requester)),
		Amount:        common.BigIntToHexStr(r.Amount),
		Outpoints:     jOutpoint,
		Receiver:      r.Receiver,
		Status:        string(r.Status),
	})
}

func (r *Redeem) UnmarshalJSON(data []byte) error {
	var jRedeem JSONRedeem
	if err := json.Unmarshal(data, &jRedeem); err != nil {
		return err
	}

	r.RequestTxHash = common.HexStrToBytes32(jRedeem.RequestTxHash)
	r.PrepareTxHash = common.HexStrToBytes32(jRedeem.PrepareTxHash)
	r.BtcTxId = common.HexStrToBytes32(jRedeem.BtcTxId)
	r.Requester = ethcommon.HexToAddress(jRedeem.Requester).Bytes()
	r.Amount = common.HexStrToBigInt(jRedeem.Amount)
	r.Receiver = jRedeem.Receiver
	r.Status = RedeemStatus(jRedeem.Status)

	for _, jOutpoint := range jRedeem.Outpoints {
		r.Outpoints = append(r.Outpoints, agreement.BtcOutpoint{
			BtcTxId: common.HexStrToBytes32(jOutpoint.BtcTxId),
			BtcIdx:  jOutpoint.BtcIdx,
		})
	}

	return nil
}

func (r *Redeem) HasPrepared() bool {
	return r.PrepareTxHash != [32]byte{} && r.Status == RedeemStatusPrepared
}

func (r *Redeem) HasCompleted() bool {
	return r.BtcTxId != [32]byte{} && r.Status == RedeemStatusCompleted
}

func (r *Redeem) IsValid() bool {
	return r.Status != RedeemStatusInvalid
}

func (r *Redeem) Clone() *Redeem {
	clone := *r
	clone.Amount = new(big.Int).Set(r.Amount)

	return &clone
}

func (r *Redeem) String() string {
	str := fmt.Sprintf("Redeem { RequestTxHash: 0x%x, PrepareTxHash: 0x%x BtcTxId: 0x%x Requester: 0x%x, Receiver: 0x%x, Amount: %v, Status: %s, ",
		r.RequestTxHash, r.PrepareTxHash, r.BtcTxId, r.Requester, r.Receiver, r.Amount, r.Status)
	str += "Outpoints: [ "
	for i, outpoint := range r.Outpoints {
		str += fmt.Sprintf("[%d]: { TxId: 0x%x, Idx: %d }, ", i, outpoint.BtcTxId, outpoint.BtcIdx)
	}
	str += " ] }"
	return str
}
