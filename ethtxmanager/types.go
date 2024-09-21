package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
)

type SignatureRequest struct {
	RequestTxHash [32]byte
	SigningHash   [32]byte
	Rx            *big.Int
	S             *big.Int
	SignatureCh   chan<- *SignatureRequest
}

type sqlSignatureRequest struct {
	RequestTxHash string
	SigningHash   string
	Rx            string
	S             string
}

func (s *SignatureRequest) convert() *sqlSignatureRequest {
	return &sqlSignatureRequest{
		RequestTxHash: common.Bytes32ToHexStr(s.RequestTxHash)[2:],
		SigningHash:   common.Bytes32ToHexStr(s.SigningHash)[2:],
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
	TxHash        [32]byte
	RequestTxHash [32]byte
	SentAt        [32]byte // hash of the latest block before sending the tx
	MinedAt       [32]byte // hash of the block where the tx is mined
}

type sqlmonitoredTx struct {
	TxHash        string
	RequestTxHash string
	SentAt        string
	MinedAt       string
}

func (mt *monitoredTx) covert() *sqlmonitoredTx {
	return &sqlmonitoredTx{
		TxHash:        common.Bytes32ToHexStr(mt.TxHash)[2:],
		RequestTxHash: common.Bytes32ToHexStr(mt.RequestTxHash)[2:],
		SentAt:        common.Bytes32ToHexStr(mt.SentAt)[2:],
		MinedAt:       common.Bytes32ToHexStr(mt.MinedAt)[2:],
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
