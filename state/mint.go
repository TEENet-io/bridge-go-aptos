package state

import (
	"fmt"
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// Mint represents the process of minting TWBTC
type Mint struct {
	BtcTxId    ethcommon.Hash // dont touch it for now
	MintTxHash ethcommon.Hash // 32 bytes
	Receiver   []byte         // address = 20 bytes on eth, 32 bytes on aptos
	Amount     *big.Int
}

// Debug function
func (m *Mint) String() string {
	return fmt.Sprintf("%+v", *m)
}

// convert MintedEvent to Mint.
func createMintFromMintedEvent(ev *agreement.MintedEvent) *Mint {
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
	Receiver   string // it stores pure hex string (from bytes), so no 0x is prepended.
	Amount     uint64
}

// encode converts Mint to sqlMint
func (s *sqlMint) encode(m *Mint) (*sqlMint, error) {
	s = &sqlMint{}
	s.BtcTxId = m.BtcTxId.String()[2:]
	s.MintTxHash = m.MintTxHash.String()[2:]
	s.Receiver = common.ByteSliceToPureHexStr(m.Receiver) // No 0x prefix!
	s.Amount = m.Amount.Uint64()

	return s, nil
}

// decode converts sqlMint to Mint
func (s *sqlMint) decode() (*Mint, error) {
	return &Mint{
		BtcTxId:    common.HexStrToBytes32("0x" + s.BtcTxId),
		MintTxHash: common.HexStrToBytes32("0x" + s.MintTxHash),
		Receiver:   common.HexStrToByteSlice(s.Receiver),
		Amount:     new(big.Int).SetUint64(s.Amount),
	}, nil
}
