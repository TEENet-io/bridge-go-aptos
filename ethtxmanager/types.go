package ethtxmanager

import (
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// This is tne enum for the status of the tx submitted to the blockchain.
type MonitoredTxStatus string

const (
	Pending  MonitoredTxStatus = "pending"
	Timeout  MonitoredTxStatus = "timeout"
	Success  MonitoredTxStatus = "success"
	Reverted MonitoredTxStatus = "reverted"
	Reorg    MonitoredTxStatus = "reorg"
)

// This is the type that Tx manager will continiously monitor
// the success/failure of Tx submitted to blockchain.
type MonitoredTx struct {
	TxHash        ethcommon.Hash
	RefIdentifier ethcommon.Hash // for redeem prepare tx: requestTxhash; for mint tx: btcTxId
	SentAfter     ethcommon.Hash // hash of the latest block before sending the tx
	SentAfterBlk  int64          // block number of the latest block before sending the tx
	MinedAt       ethcommon.Hash // hash of the block where the tx is mined
	Status        MonitoredTxStatus
}

// Store in SQLite
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
	s.Id = mt.RefIdentifier.String()[2:]
	s.SentAfter = mt.SentAfter.String()[2:]
	s.SentAfterBlk = mt.SentAfterBlk
	s.MinedAt = mt.MinedAt.String()[2:]
	s.Status = string(mt.Status)

	return s
}

func (s *sqlMonitoredTx) decode() *MonitoredTx {
	return &MonitoredTx{
		TxHash:        common.HexStrToBytes32(s.TxHash),
		RefIdentifier: common.HexStrToBytes32(s.Id),
		SentAfter:     common.HexStrToBytes32(s.SentAfter),
		SentAfterBlk:  s.SentAfterBlk,
		MinedAt:       common.HexStrToBytes32(s.MinedAt),
		Status:        MonitoredTxStatus(s.Status),
	}
}
