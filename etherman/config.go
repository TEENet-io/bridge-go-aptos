package etherman

import "github.com/ethereum/go-ethereum/common"

type Config struct {
	URL string

	BridgeContractAddress common.Address
	TWBTCContractAddress  common.Address
}
