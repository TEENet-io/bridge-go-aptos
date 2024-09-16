package eth2btcstate

import (
	"encoding/json"
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

type Redeem struct {
	RequestTxHash [32]byte
	PrepareTxHash [32]byte
	BtcTxId       [32]byte
	Requester     ethcommon.Address
	Amount        *big.Int
	Outpoints     []Outpoint
	Receiver      string // receiver btc address
	Status        RedeemStatus
}

func (r *Redeem) SetFromRequestedEvent(ev *ethsync.RedeemRequestedEvent) *Redeem {
	r.RequestTxHash = ev.RedeemRequestTxHash
	r.Requester = ev.Requester
	r.Amount = new(big.Int).Set(ev.Amount)
	r.Receiver = ev.Receiver
	if ev.IsValidReceiver {
		r.Status = RedeemStatusRequested
	} else {
		r.Status = RedeemStatusInvalid
	}

	return r
}

func (r *Redeem) SetFromPreparedEvent(ev *ethsync.RedeemPreparedEvent) *Redeem {
	r.PrepareTxHash = ev.RedeemPrepareTxHash
	for i := range ev.OutpointIdxs {
		r.Outpoints = append(r.Outpoints, Outpoint{
			TxId: ev.OutpointTxIds[i],
			Idx:  ev.OutpointIdxs[i],
		})
	}

	r.Status = RedeemStatusPrepared

	return r
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
	clone := &Redeem{}
	clone.RequestTxHash = r.RequestTxHash
	clone.PrepareTxHash = r.PrepareTxHash
	clone.BtcTxId = r.BtcTxId
	clone.Requester = r.Requester
	clone.Amount = new(big.Int).Set(r.Amount)
	clone.Outpoints = make([]Outpoint, len(r.Outpoints))
	clone.Receiver = r.Receiver
	copy(clone.Outpoints, r.Outpoints)

	return clone
}

func (r *Redeem) Equal(other *Redeem) bool {
	if r.RequestTxHash != other.RequestTxHash {
		return false
	}

	if r.PrepareTxHash != other.PrepareTxHash {
		return false
	}

	if r.BtcTxId != other.BtcTxId {
		return false
	}

	if r.Requester.Hex() != other.Requester.Hex() {
		return false
	}

	if r.Amount.Cmp(other.Amount) != 0 {
		return false
	}

	if len(r.Outpoints) != len(other.Outpoints) {
		return false
	}

	for i := range r.Outpoints {
		if r.Outpoints[i].TxId != other.Outpoints[i].TxId {
			return false
		}

		if r.Outpoints[i].Idx != other.Outpoints[i].Idx {
			return false
		}
	}

	return true
}
