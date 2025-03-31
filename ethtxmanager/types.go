package ethtxmanager

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type SignatureRequest struct {
	Id          ethcommon.Hash
	SigningHash ethcommon.Hash
	Outpoints   []state.BtcOutpoint
	Rx          *big.Int
	S           *big.Int
}

type MonitoredTxStatus string

const (
	Pending  MonitoredTxStatus = "pending"
	Timeout  MonitoredTxStatus = "timeout"
	Success  MonitoredTxStatus = "success"
	Reverted MonitoredTxStatus = "reverted"
	Reorg    MonitoredTxStatus = "reorg"
)

type MonitoredTx struct {
	TxHash       ethcommon.Hash
	Id           ethcommon.Hash // requestTxhash for redeem prepare tx and btcTxId for mint tx
	SentAfter    ethcommon.Hash // hash of the latest block before sending the tx
	SentAfterBlk int64          // block number of the latest block before sending the tx
	MinedAt      ethcommon.Hash // hash of the block where the tx is mined
	Status       MonitoredTxStatus
}

type sqlMonitoredTx struct {
	TxHash       string
	Id           string
	SentAfter    string
	SentAfterBlk int64
	MinedAt      string
	Status       string
}

func (s *sqlMonitoredTx) encode(mt *MonitoredTx) *sqlMonitoredTx {
	s.TxHash = mt.TxHash.String()[2:]
	s.Id = mt.Id.String()[2:]
	s.SentAfter = mt.SentAfter.String()[2:]
	s.SentAfterBlk = mt.SentAfterBlk
	s.MinedAt = mt.MinedAt.String()[2:]
	s.Status = string(mt.Status)

	return s
}

func (s *sqlMonitoredTx) decode() *MonitoredTx {
	return &MonitoredTx{
		TxHash:       common.HexStrToBytes32(s.TxHash),
		Id:           common.HexStrToBytes32(s.Id),
		SentAfter:    common.HexStrToBytes32(s.SentAfter),
		SentAfterBlk: s.SentAfterBlk,
		MinedAt:      common.HexStrToBytes32(s.MinedAt),
		Status:       MonitoredTxStatus(s.Status),
	}
}
