package btc2ethstate

import (
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type MintStatus string

const (
	MintStatusRequested MintStatus = "requested"
	MintStatusCompleted MintStatus = "completed"
)

type Outpoint struct {
	TxId ethcommon.Hash
	Idx  uint16
}

// Mint represents the process of minting TWBTC
type Mint struct {
	BtcTxID    ethcommon.Hash
	MintTxHash ethcommon.Hash
	Receiver   ethcommon.Address
	Amount     *big.Int
	Outpoints  []state.Outpoint
	Status     MintStatus
}

func (m *Mint) String() string {
	return fmt.Sprintf("%+v", *m)
}

type sqlMint struct {
	BtcTxID    string
	MintTxHash string
	Receiver   string
	Amount     uint64
	Outpoints  []byte
	Status     string
}

func encode(m *Mint) (s *sqlMint, err error) {
	outpoints, err := state.EncodeOutpoints(m.Outpoints)
	if err != nil {
		return nil, err
	}

	s = &sqlMint{}
	s.BtcTxID = m.BtcTxID.String()[2:]
	s.MintTxHash = m.MintTxHash.String()[2:]
	s.Receiver = m.Receiver.String()[2:]
	s.Amount = m.Amount.Uint64()
	s.Outpoints = outpoints
	s.Status = string(m.Status)

	return
}

func (s *sqlMint) decode() (*Mint, error) {
	outpoints, err := state.DecodeOutpoints(s.Outpoints)
	if err != nil {
		return nil, err
	}

	return &Mint{
		BtcTxID:    common.HexStrToBytes32("0x" + s.BtcTxID),
		MintTxHash: common.HexStrToBytes32("0x" + s.MintTxHash),
		Receiver:   ethcommon.HexToAddress("0x" + s.Receiver),
		Amount:     new(big.Int).SetUint64(s.Amount),
		Outpoints:  outpoints,
		Status:     MintStatus(s.Status),
	}, nil
}
