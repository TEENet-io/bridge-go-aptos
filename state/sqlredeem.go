package state

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type sqlRedeem struct {
	RequestTxHash string
	PrepareTxHash string
	BtcTxId       string
	Requester     string
	Receiver      string
	Amount        uint64
	Outpoints     []byte
	Status        string
}

// encode converts fields of an eth2btcRedeem object to
// relevant types that can be stored in sql db. It only checks field
// Outpoints for non-emptyness since it is difficult to do the check
// in db level.
func (s *sqlRedeem) encode(r *Redeem) (*sqlRedeem, error) {
	outpoints, err := encodeOutpoints(r.Outpoints)
	if err != nil {
		return nil, err
	}

	s.RequestTxHash = r.RequestTxHash.String()[2:]
	s.PrepareTxHash = r.PrepareTxHash.String()[2:]
	s.BtcTxId = r.BtcTxId.String()[2:]
	s.Requester = r.Requester.String()[2:]
	s.Receiver = r.Receiver
	s.Amount = r.Amount.Uint64()
	s.Outpoints = outpoints
	s.Status = string(r.Status)

	return s, nil
}

func (r *sqlRedeem) decode() (*Redeem, error) {
	requestTxHash := common.HexStrToBytes32("0x" + r.RequestTxHash)
	prepareTxHash := common.HexStrToBytes32("0x" + r.PrepareTxHash)
	btcTxId := common.HexStrToBytes32("0x" + r.BtcTxId)
	requester := ethcommon.HexToAddress("0x" + r.Requester)
	amount := new(big.Int).SetUint64(r.Amount)

	outpoints, err := decodeOutpoints(r.Outpoints)
	if err != nil {
		return nil, err
	}

	return &Redeem{
		RequestTxHash: requestTxHash,
		PrepareTxHash: prepareTxHash,
		BtcTxId:       btcTxId,
		Requester:     requester,
		Receiver:      r.Receiver,
		Amount:        amount,
		Outpoints:     outpoints,
		Status:        RedeemStatus(r.Status),
	}, nil
}
