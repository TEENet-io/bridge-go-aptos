package btcwallet

import (
	"math/big"

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
