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

// type sqlSignatureRequest struct {
// 	RequestTxHash string
// 	SigningHash   string
// 	Outpoints     []byte
// 	Rx            string
// 	S             string
// }

// func (s *sqlSignatureRequest) encode(sr *SignatureRequest) (*sqlSignatureRequest, error) {
// 	s.RequestTxHash = sr.RequestTxHash.String()[2:]
// 	s.SigningHash = sr.SigningHash.String()[2:]
// 	s.Rx = common.BigIntToHexStr(sr.Rx)[2:]
// 	s.S = common.BigIntToHexStr(sr.S)[2:]

// 	outpoints, err := state.EncodeOutpoints(sr.Outpoints)
// 	if err != nil {
// 		return nil, err
// 	}
// 	s.Outpoints = append([]byte{}, outpoints...)

// 	return s, nil
// }

// func (s *sqlSignatureRequest) decode() (*SignatureRequest, error) {
// 	outpoints, err := state.DecodeOutpoints(s.Outpoints)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &SignatureRequest{
// 		RequestTxHash: common.HexStrToBytes32(s.RequestTxHash),
// 		SigningHash:   common.HexStrToBytes32(s.SigningHash),
// 		Outpoints:     append([]state.Outpoint{}, outpoints...),
// 		Rx:            common.HexStrToBigInt(s.Rx),
// 		S:             common.HexStrToBigInt(s.S),
// 	}, nil
// }

type MonitoredTxStatus string

const (
	Pending  MonitoredTxStatus = "pending"
	Timeout  MonitoredTxStatus = "timeout"
	Success  MonitoredTxStatus = "success"
	Reverted MonitoredTxStatus = "reverted"
	Reorg    MonitoredTxStatus = "reorg"
)

type MonitoredTx struct {
	TxHash      ethcommon.Hash
	Id          ethcommon.Hash // requestTxhash for redeem prepare tx and btcTxId for mint tx
	SigningHash ethcommon.Hash
	Outpoints   []state.Outpoint
	Rx          *big.Int
	S           *big.Int
	SentAfter   ethcommon.Hash // hash of the latest block before sending the tx
	MinedAt     ethcommon.Hash // hash of the block where the tx is mined
	Status      MonitoredTxStatus
}

type sqlMonitoredTx struct {
	TxHash      string
	Id          string
	SigningHash string
	Outpoints   []byte
	Rx          string
	S           string
	SentAfter   string
	MinedAt     string
	Status      string
}

func (s *sqlMonitoredTx) encode(mt *MonitoredTx) (*sqlMonitoredTx, error) {
	s.TxHash = mt.TxHash.String()[2:]
	s.Id = mt.Id.String()[2:]
	s.SigningHash = mt.SigningHash.String()[2:]
	s.Rx = common.BigIntToHexStr(mt.Rx)[2:]
	s.S = common.BigIntToHexStr(mt.S)[2:]
	s.SentAfter = mt.SentAfter.String()[2:]
	s.MinedAt = mt.MinedAt.String()[2:]
	s.Status = string(mt.Status)

	outpoints, err := state.EncodeOutpoints(mt.Outpoints)
	if err != nil {
		return nil, err
	}
	s.Outpoints = append([]byte{}, outpoints...)

	return s, nil
}

func (s *sqlMonitoredTx) decode() (*MonitoredTx, error) {
	outpoints, err := state.DecodeOutpoints(s.Outpoints)
	if err != nil {
		return nil, err
	}

	return &MonitoredTx{
		TxHash:      common.HexStrToBytes32(s.TxHash),
		Id:          common.HexStrToBytes32(s.Id),
		SigningHash: common.HexStrToBytes32(s.SigningHash),
		Outpoints:   append([]state.Outpoint{}, outpoints...),
		Rx:          common.HexStrToBigInt(s.Rx),
		S:           common.HexStrToBigInt(s.S),
		SentAfter:   common.HexStrToBytes32(s.SentAfter),
		MinedAt:     common.HexStrToBytes32(s.MinedAt),
		Status:      MonitoredTxStatus(s.Status),
	}, nil
}
