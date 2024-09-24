package state

import (
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type MintStatus string

const (
	MintStatusRequested MintStatus = "requested"
	MintStatusCompleted MintStatus = "completed"
)

// Mint represents the process of minting TWBTC
type Mint struct {
	BtcTxID    ethcommon.Hash
	MintTxHash ethcommon.Hash
	Receiver   ethcommon.Address
	Amount     *big.Int
	Status     MintStatus
}

func (m *Mint) String() string {
	return fmt.Sprintf("%+v", *m)
}

func createMintFromMintedEvent(ev *ethsync.MintedEvent) *Mint {
	return &Mint{
		BtcTxID:    ev.BtcTxId,
		MintTxHash: ev.MintTxHash,
		Receiver:   ev.Receiver,
		Amount:     ev.Amount,
		Status:     MintStatusRequested,
	}
}

type sqlMint struct {
	BtcTxID    string
	MintTxHash string
	Receiver   string
	Amount     uint64
	Status     string
}

func (s *sqlMint) encode(m *Mint) (*sqlMint, error) {
	s = &sqlMint{}
	s.BtcTxID = m.BtcTxID.String()[2:]
	s.MintTxHash = m.MintTxHash.String()[2:]
	s.Receiver = m.Receiver.String()[2:]
	s.Amount = m.Amount.Uint64()
	s.Status = string(m.Status)

	return s, nil
}

func (s *sqlMint) decode() (*Mint, error) {
	return &Mint{
		BtcTxID:    common.HexStrToBytes32("0x" + s.BtcTxID),
		MintTxHash: common.HexStrToBytes32("0x" + s.MintTxHash),
		Receiver:   ethcommon.HexToAddress("0x" + s.Receiver),
		Amount:     new(big.Int).SetUint64(s.Amount),
		Status:     MintStatus(s.Status),
	}, nil
}
