package state

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

type Outpoint struct {
	TxId ethcommon.Hash
	Idx  uint16
}
