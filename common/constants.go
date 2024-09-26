package common

import (
	ethcommon "github.com/ethereum/go-ethereum/common"
)

var (
	EmptyHash = ethcommon.BytesToHash([]byte{})
)
