package etherman

import "github.com/ethereum/go-ethereum/common"

type Config struct {
	// URL is the URL of the Ethereum node
	URL string

	// BridgeContractAddress is the deployed bridge contract address in hex string
	BridgeContractAddress common.Address
}
