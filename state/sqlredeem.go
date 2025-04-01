package state

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
)

type sqlRedeem struct {
	RequestTxHash string
	PrepareTxHash string
	BtcTxId       string
	Requester     string // hex representation of address (no 0x prefix)
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
	outpoints, err := EncodeOutpoints(r.Outpoints)
	if err != nil {
		return nil, err
	}

	s.RequestTxHash = r.RequestTxHash.String()[2:]
	s.PrepareTxHash = r.PrepareTxHash.String()[2:]
	// btc txid is 32bytes long, but it doesn't usually prefix with 0x
	s.BtcTxId = r.BtcTxId.String()[2:]
	s.Requester = common.ByteSliceToPureHexStr(r.Requester)
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
	requester := common.HexStrToByteSlice(r.Requester)
	amount := new(big.Int).SetUint64(r.Amount)

	outpoints, err := DecodeOutpoints(r.Outpoints)
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
