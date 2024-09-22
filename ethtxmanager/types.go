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

func (s *SignatureRequest) convert() *sqlSignatureRequest {
	return &sqlSignatureRequest{
		RequestTxHash: s.RequestTxHash.String()[2:],
		SigningHash:   s.SigningHash.String()[2:],
		Rx:            common.BigIntToHexStr(s.Rx)[2:],
		S:             common.BigIntToHexStr(s.S)[2:],
	}
}

func (s *SignatureRequest) restore(sqlSr *sqlSignatureRequest) *SignatureRequest {
	s = &SignatureRequest{
		RequestTxHash: common.HexStrToBytes32(sqlSr.RequestTxHash),
		SigningHash:   common.HexStrToBytes32(sqlSr.SigningHash),
		Rx:            common.HexStrToBigInt(sqlSr.Rx),
		S:             common.HexStrToBigInt(sqlSr.S),
	}
	return s
}

type monitoredTx struct {
	TxHash        ethcommon.Hash
	RequestTxHash ethcommon.Hash
	SentAt        ethcommon.Hash // hash of the latest block before sending the tx
	MinedAt       ethcommon.Hash // hash of the block where the tx is mined
}

type sqlmonitoredTx struct {
	TxHash        string
	RequestTxHash string
	SentAt        string
	MinedAt       string
}

func (mt *monitoredTx) covert() *sqlmonitoredTx {
	return &sqlmonitoredTx{
		TxHash:        mt.TxHash.String()[2:],
		RequestTxHash: mt.RequestTxHash.String()[2:],
		SentAt:        mt.SentAt.String()[2:],
		MinedAt:       mt.MinedAt.String()[2:],
	}
}

func (mt *monitoredTx) restore(sqlMt *sqlmonitoredTx) *monitoredTx {
	mt = &monitoredTx{
		TxHash:        common.HexStrToBytes32(sqlMt.TxHash),
		RequestTxHash: common.HexStrToBytes32(sqlMt.RequestTxHash),
		SentAt:        common.HexStrToBytes32(sqlMt.SentAt),
		MinedAt:       common.HexStrToBytes32(sqlMt.MinedAt),
	}
	return mt
}
