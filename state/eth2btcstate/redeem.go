package eth2btcstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type Outpoint struct {
	TxId [32]byte
	Idx  uint16
}

type RedeemStatus string

const (
	RedeemStatusRequested RedeemStatus = "requested"
	RedeemStatusPrepared  RedeemStatus = "prepared"
	RedeemStatusCompleted RedeemStatus = "completed"
	RedeemStatusInvalid   RedeemStatus = "invalid"
)

var (
	ErrorAmountInvalid              = "amount invalid"
	ErrorRequesterInvalid           = "requester address invalid"
	ErrorRequestTxHashInvalid       = "redeem request tx hash invalid"
	ErrorRedeemPrepareTxHashInvalid = "redeem prepare tx hash invalid"
	ErrorRequestTxHashUnmatched     = "redeem request tx hash unmatched"
	ErrorRequesterUnmatched         = "requester unmatched"
	ErrorAmountUnmatched            = "amount unmatched"
	ErrorReceiverUnmatched          = "receiver unmatched"
	ErrorOutpointsInvalid           = "outpoints invalid"
	ErrorOutpointTxIdInvalid        = "outpoint tx id invalid"
	ErrorRequireStatusRequested     = "require status == requested"
)

type Redeem struct {
	RequestTxHash [32]byte
	PrepareTxHash [32]byte
	BtcTxId       [32]byte
	Requester     ethcommon.Address
	Receiver      string // receiver btc address
	Amount        *big.Int
	Outpoints     []Outpoint
	Status        RedeemStatus
}

func createRedeemFromRequestedEvent(ev *ethsync.RedeemRequestedEvent) (*Redeem, error) {
	r := &Redeem{}

	if ev.RequestTxHash == [32]byte{} {
		return nil, errors.New(ErrorRequestTxHashInvalid)
	}

	if ev.Requester == [20]byte{} {
		return nil, errors.New(ErrorRequesterInvalid)
	}

	if ev.Amount == nil || ev.Amount.Sign() <= 0 {
		return nil, errors.New(ErrorAmountInvalid)
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

func (r *Redeem) updateFromPreparedEvent(ev *ethsync.RedeemPreparedEvent) (*Redeem, error) {
	if r.Status != RedeemStatusRequested {
		return nil, errors.New(ErrorRequireStatusRequested)
	}

	if ev.PrepareTxHash == [32]byte{} {
		return nil, errors.New(ErrorRedeemPrepareTxHashInvalid)
	}

	if ev.RequestTxHash != r.RequestTxHash {
		return nil, errors.New(ErrorRequestTxHashUnmatched)
	}

	if ev.Requester != r.Requester {
		return nil, errors.New(ErrorRequesterUnmatched)
	}

	if ev.Amount == nil {
		return nil, errors.New(ErrorAmountInvalid)
	}

	if ev.Amount.Cmp(r.Amount) != 0 {
		return nil, errors.New(ErrorAmountUnmatched)
	}

	if ev.Receiver != r.Receiver {
		return nil, errors.New(ErrorReceiverUnmatched)
	}

	if ev.OutpointTxIds == nil || ev.OutpointIdxs == nil || len(ev.OutpointTxIds) == 0 || len(ev.OutpointIdxs) == 0 {
		return nil, errors.New(ErrorOutpointsInvalid)
	}

	r.PrepareTxHash = ev.PrepareTxHash
	for i := range ev.OutpointIdxs {
		if ev.OutpointTxIds[i] == [32]byte{} {
			return nil, errors.New(ErrorOutpointTxIdInvalid)
		}

		r.Outpoints = append(r.Outpoints, Outpoint{
			TxId: ev.OutpointTxIds[i],
			Idx:  ev.OutpointIdxs[i],
		})
	}

	r.Status = RedeemStatusPrepared

	return r, nil
}

func createRedeemFromPreparedEvent(ev *ethsync.RedeemPreparedEvent) (*Redeem, error) {
	r, err := createRedeemFromRequestedEvent(&ethsync.RedeemRequestedEvent{
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
	jOutpoint := []JSONOutpoint{}
	for _, outpoint := range r.Outpoints {
		jOutpoint = append(jOutpoint, JSONOutpoint{
			TxId: common.Bytes32ToHexStr(outpoint.TxId),
			Idx:  outpoint.Idx,
		})
	}

	return json.Marshal(&JSONRedeem{
		RequestTxHash: common.Bytes32ToHexStr(r.RequestTxHash),
		PrepareTxHash: common.Bytes32ToHexStr(r.PrepareTxHash),
		BtcTxId:       common.Bytes32ToHexStr(r.BtcTxId),
		Requester:     r.Requester.Hex(),
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
	r.Requester = ethcommon.HexToAddress(jRedeem.Requester)
	r.Amount = common.HexStrToBigInt(jRedeem.Amount)
	r.Receiver = jRedeem.Receiver
	r.Status = RedeemStatus(jRedeem.Status)

	for _, jOutpoint := range jRedeem.Outpoints {
		r.Outpoints = append(r.Outpoints, Outpoint{
			TxId: common.HexStrToBytes32(jOutpoint.TxId),
			Idx:  jOutpoint.Idx,
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
		r.RequestTxHash, r.PrepareTxHash, r.BtcTxId, r.Requester.Hex(), r.Receiver, r.Amount, r.Status)
	str += "Outpoints: [ "
	for i, outpoint := range r.Outpoints {
		str += fmt.Sprintf("[%d]: { TxId: 0x%x, Idx: %d }, ", i, outpoint.TxId, outpoint.Idx)
	}
	str += " ] }"
	return str
}
