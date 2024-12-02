package etherman

import "github.com/ethereum/go-ethereum/common"

type Config struct {
	URL string // URL of ETH RPC Node

	BridgeContractAddress common.Address
	TWBTCContractAddress  common.Address
}
