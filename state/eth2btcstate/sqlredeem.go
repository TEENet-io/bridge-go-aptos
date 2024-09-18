package eth2btcstate

import (
	"bytes"
	"encoding/gob"
	"errors"
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

// encodeRedeem converts fields of an eth2btcstate.Redeem object to
// relevant types that can be stored in sql db. It only checks field
// Outpoints for non-emptyness since it is difficult to do the check
// in db level.
func encode(r *Redeem) (*sqlRedeem, error) {
	outpoints, err := encodeOutpoints(r.Outpoints)
	if err != nil {
		return nil, err
	}

	return &sqlRedeem{
		RequestTxHash: common.Bytes32ToHexStr(r.RequestTxHash)[2:],
		PrepareTxHash: common.Bytes32ToHexStr(r.PrepareTxHash)[2:],
		BtcTxId:       common.Bytes32ToHexStr(r.BtcTxId)[2:],
		Requester:     r.Requester.String()[2:],
		Receiver:      r.Receiver,
		Amount:        r.Amount.Uint64(),
		Outpoints:     outpoints,
		Status:        string(r.Status),
	}, nil
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

func encodeOutpoints(outpoints []Outpoint) ([]byte, error) {
	if outpoints == nil {
		return nil, nil
	}

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(outpoints); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeOutpoints(data []byte) ([]Outpoint, error) {
	if data == nil {
		return nil, nil
	}

	if len(data) == 0 {
		return nil, errors.New("expect non-empty bytes")
	}

	decoder := gob.NewDecoder(bytes.NewReader(data))
	var outpoints []Outpoint
	if err := decoder.Decode(&outpoints); err != nil {
		return nil, err
	}

	return outpoints, nil
}
