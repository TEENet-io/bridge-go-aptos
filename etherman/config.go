package etherman

type Config struct {
	// URL is the URL of the Ethereum node
	URL string

	// BridgeContractAddress is the deployed bridge contract address in hex string
	BridgeContractAddress string
}
