package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type SignatureRequest struct {
	RequestTxHash ethcommon.Hash
	SigningHash   ethcommon.Hash
	Rx            *big.Int
	S             *big.Int
}

type sqlSignatureRequest struct {
	RequestTxHash string
	SigningHash   string
	Rx            string
	S             string
}

func (s *sqlSignatureRequest) encode(sr *SignatureRequest) *sqlSignatureRequest {
	s.RequestTxHash = sr.RequestTxHash.String()[2:]
	s.SigningHash = sr.SigningHash.String()[2:]
	s.Rx = common.BigIntToHexStr(sr.Rx)[2:]
	s.S = common.BigIntToHexStr(sr.S)[2:]
	return s
}

func (s *sqlSignatureRequest) decode() *SignatureRequest {
	return &SignatureRequest{
		RequestTxHash: common.HexStrToBytes32(s.RequestTxHash),
		SigningHash:   common.HexStrToBytes32(s.SigningHash),
		Rx:            common.HexStrToBigInt(s.Rx),
		S:             common.HexStrToBigInt(s.S),
	}
}

type monitoredTx struct {
	TxHash        ethcommon.Hash
	RequestTxHash ethcommon.Hash
	SentAfter     ethcommon.Hash // hash of the latest block before sending the tx
}

type sqlMonitoredTx struct {
	TxHash        string
	RequestTxHash string
	SentAfter     string
}

func (s *sqlMonitoredTx) encode(mt *monitoredTx) *sqlMonitoredTx {
	s.TxHash = mt.TxHash.String()[2:]
	s.RequestTxHash = mt.RequestTxHash.String()[2:]
	s.SentAfter = mt.SentAfter.String()[2:]

	return s
}

func (s *sqlMonitoredTx) decode() *monitoredTx {
	return &monitoredTx{
		TxHash:        common.HexStrToBytes32(s.TxHash),
		RequestTxHash: common.HexStrToBytes32(s.RequestTxHash),
		SentAfter:     common.HexStrToBytes32(s.SentAfter),
	}
}
