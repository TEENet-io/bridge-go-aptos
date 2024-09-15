package common

import (
	"crypto/rand"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

func RandEthAddress() ethcommon.Address {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return ethcommon.Address{}
	}
	return ethcommon.BytesToAddress(b[:])
}
