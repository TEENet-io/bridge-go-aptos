package state

import (
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/ethsync"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// Mint represents the process of minting TWBTC
type Mint struct {
	BtcTxId    ethcommon.Hash
	MintTxHash ethcommon.Hash
	Receiver   ethcommon.Address
	Amount     *big.Int
}

func (m *Mint) String() string {
	return fmt.Sprintf("%+v", *m)
}

// convert MintedEvent to Mint.
func createMintFromMintedEvent(ev *ethsync.MintedEvent) *Mint {
	return &Mint{
		BtcTxId:    ev.BtcTxId,
		MintTxHash: ev.MintTxHash,
		Receiver:   ev.Receiver,
		Amount:     ev.Amount,
	}
}

type sqlMint struct {
	BtcTxId    string
	MintTxHash string
	Receiver   string
	Amount     uint64
}

// encode converts Mint to sqlMint
func (s *sqlMint) encode(m *Mint) (*sqlMint, error) {
	s = &sqlMint{}
	s.BtcTxId = m.BtcTxId.String()[2:]
	s.MintTxHash = m.MintTxHash.String()[2:]
	s.Receiver = m.Receiver.String()[2:]
	s.Amount = m.Amount.Uint64()

	return s, nil
}

// decode converts sqlMint to Mint
func (s *sqlMint) decode() (*Mint, error) {
	return &Mint{
		BtcTxId:    common.HexStrToBytes32("0x" + s.BtcTxId),
		MintTxHash: common.HexStrToBytes32("0x" + s.MintTxHash),
		Receiver:   ethcommon.HexToAddress("0x" + s.Receiver),
		Amount:     new(big.Int).SetUint64(s.Amount),
	}, nil
}
