package eth2btcstate

import (
	"encoding/json"
	"errors"
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
	ErrorAmountInvalid                = "amount invalid"
	ErrorRequesterInvalid             = "requester address invalid"
	ErrorRedeemRequestTxHashInvalid   = "redeem request tx hash invalid"
	ErrorRedeemPrepareTxHashInvalid   = "redeem prepare tx hash invalid"
	ErrorRedeemRequestTxHashUnmatched = "redeem request tx hash unmatched"
	ErrorRequesterUnmatched           = "requester unmatched"
	ErrorAmountUnmatched              = "amount unmatched"
	ErrorReceiverUnmatched            = "receiver unmatched"
	ErrorOutpointsInvalid             = "outpoints invalid"
	ErrorOutpointTxIdInvalid          = "outpoint tx id invalid"
	ErrorRequireStatusRequested       = "require status == requested"
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

func createFromRequestedEvent(ev *ethsync.RedeemRequestedEvent) (*Redeem, error) {
	r := &Redeem{}

	if ev.RedeemRequestTxHash == [32]byte{} {
		return nil, errors.New(ErrorRedeemRequestTxHashInvalid)
	}

	if ev.Requester == [20]byte{} {
		return nil, errors.New(ErrorRequesterInvalid)
	}

	if ev.Amount == nil || ev.Amount.Sign() <= 0 {
		return nil, errors.New(ErrorAmountInvalid)
	}

	r.RequestTxHash = ev.RedeemRequestTxHash
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

	if ev.RedeemPrepareTxHash == [32]byte{} {
		return nil, errors.New(ErrorRedeemPrepareTxHashInvalid)
	}

	if ev.RedeemRequestTxHash != r.RequestTxHash {
		return nil, errors.New(ErrorRedeemRequestTxHashUnmatched)
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

	r.PrepareTxHash = ev.RedeemPrepareTxHash
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
	j, _ := r.MarshalJSON()
	return string(j)
}
