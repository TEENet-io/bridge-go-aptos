package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type SignatureRequest struct {
	RequestTxHash ethcommon.Hash
	SigningHash   ethcommon.Hash
	Outpoints     []state.Outpoint
	Rx            *big.Int
	S             *big.Int
}

type sqlSignatureRequest struct {
	RequestTxHash string
	SigningHash   string
	Outpoints     []byte
	Rx            string
	S             string
}

func (s *sqlSignatureRequest) encode(sr *SignatureRequest) (*sqlSignatureRequest, error) {
	s.RequestTxHash = sr.RequestTxHash.String()[2:]
	s.SigningHash = sr.SigningHash.String()[2:]
	s.Rx = common.BigIntToHexStr(sr.Rx)[2:]
	s.S = common.BigIntToHexStr(sr.S)[2:]

	outpoints, err := state.EncodeOutpoints(sr.Outpoints)
	if err != nil {
		return nil, err
	}
	s.Outpoints = append([]byte{}, outpoints...)

	return s, nil
}

func (s *sqlSignatureRequest) decode() (*SignatureRequest, error) {
	outpoints, err := state.DecodeOutpoints(s.Outpoints)
	if err != nil {
		return nil, err
	}

	return &SignatureRequest{
		RequestTxHash: common.HexStrToBytes32(s.RequestTxHash),
		SigningHash:   common.HexStrToBytes32(s.SigningHash),
		Outpoints:     append([]state.Outpoint{}, outpoints...),
		Rx:            common.HexStrToBigInt(s.Rx),
		S:             common.HexStrToBigInt(s.S),
	}, nil
}

type monitoredTx struct {
	TxHash    ethcommon.Hash
	Id        ethcommon.Hash // requestTxhash for redeem prepare tx and btcTxId for mint tx
	SentAfter ethcommon.Hash // hash of the latest block before sending the tx
}

type sqlMonitoredTx struct {
	TxHash    string
	Id        string
	SentAfter string
}

func (s *sqlMonitoredTx) encode(mt *monitoredTx) *sqlMonitoredTx {
	s.TxHash = mt.TxHash.String()[2:]
	s.Id = mt.Id.String()[2:]
	s.SentAfter = mt.SentAfter.String()[2:]

	return s
}

func (s *sqlMonitoredTx) decode() *monitoredTx {
	return &monitoredTx{
		TxHash:    common.HexStrToBytes32(s.TxHash),
		Id:        common.HexStrToBytes32(s.Id),
		SentAfter: common.HexStrToBytes32(s.SentAfter),
	}
}
