package btcwallet

import (
	"math/big"
	"time"

	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type Spendable struct {
	BtcTxId     ethcommon.Hash
	Idx         uint16
	Amount      *big.Int
	BlockNumber *big.Int
	Lock        bool
}

type sqlSpendable struct {
	BtcTxId     string
	Idx         uint16 // to be saved as INT in db
	Amount      int64  // to be saved as BIGINT in db
	BlockNumber int64  // to be saved as BIGINT in db
	Lock        bool
}

func (s *sqlSpendable) encode(sp *Spendable) (*sqlSpendable, error) {
	s.BtcTxId = sp.BtcTxId.String()[2:]
	s.Idx = sp.Idx
	s.Amount = sp.Amount.Int64()
	s.BlockNumber = sp.BlockNumber.Int64()
	s.Lock = sp.Lock

	return s, nil
}

func (s *sqlSpendable) decode() (*Spendable, error) {
	btcTxId := ethcommon.HexToHash("0x" + s.BtcTxId)
	amount := new(big.Int).SetInt64(s.Amount)
	blockNumber := new(big.Int).SetInt64(s.BlockNumber)
	return &Spendable{
		BtcTxId:     btcTxId,
		Idx:         s.Idx,
		Amount:      amount,
		BlockNumber: blockNumber,
		Lock:        s.Lock,
	}, nil
}

type RequestStatus string

const (
	Locked  RequestStatus = "locked"
	Timeout RequestStatus = "timeout"
	Spent   RequestStatus = "spent"
)

type Request struct {
	Id        ethcommon.Hash
	Outpoints []state.Outpoint
	CreatedAt time.Time
	Status    RequestStatus
}

type sqlRequest struct {
	Id        string
	Outpoints []byte
	CreatedAt time.Time
	Status    string
}

func (r *sqlRequest) encode(req *Request) (*sqlRequest, error) {
	outpoints, err := state.EncodeOutpoints(req.Outpoints)
	if err != nil {
		return nil, err
	}

	r.Id = req.Id.String()[2:]
	r.Outpoints = outpoints
	// r.CreatedAt = req.CreatedAt.Unix()
	r.Status = string(req.Status)
	return r, nil
}

func (r *sqlRequest) decode() (*Request, error) {
	id := ethcommon.HexToHash("0x" + r.Id)
	outpoints, err := state.DecodeOutpoints(r.Outpoints)
	if err != nil {
		return nil, err
	}

	return &Request{
		Id:        id,
		Outpoints: outpoints,
		CreatedAt: r.CreatedAt,
		Status:    RequestStatus(r.Status),
	}, nil
}
