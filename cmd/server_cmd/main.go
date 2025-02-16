package main

import (
	"fmt"

	"github.com/TEENet-io/bridge-go/multisig"
	"github.com/btcsuite/btcd/chaincfg"

	"github.com/TEENet-io/bridge-go/cmd"
	"github.com/spf13/viper"
)

const (
	ENV_CONFIG_FILE_PATH = "BRIDGE_CONFIG"
)

func main() {
	// Set overall config level to Debug
	// logconfig.ConfigDebugLogger()

	// Tool to read environment variables
	viper.AutomaticEnv()

	// Accessing an environment variable of configuration file location.
	_config_file := viper.GetString(ENV_CONFIG_FILE_PATH)
	fmt.Printf("Bridge server configuration file = %s\n", _config_file)

	// See if file exists
	if !cmd.FileExists(_config_file) {
		fmt.Printf("Bridge server configuration file not found: %s\n", _config_file)
		return
	}

	// Read from config file.
	success := initializeViper(_config_file)
	if !success {
		return
	}

	// Make the configuration
	bsc := PrepareBridgeServerConfig()
	if bsc == nil {
		fmt.Printf("Error loading bridge server configuration\n")
		return
	}

	fmt.Println("Starting bridge server... press Ctrl+C to kill the server")
	// Start server and block.
	cmd.StartBridgeServerAndWait(bsc)
}

func initializeViper(filePath string) bool {
	viper.SetConfigFile(filePath)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading configuration file, %s", err)
		return false
	}
	return true
}

// PrepareBridgeServerConfig reads configuration variables and returns a BridgeServerConfig.
func PrepareBridgeServerConfig() *cmd.BridgeServerConfig {

	// *** prepare objects that aren't string type ***

	// Parse the BTC chain config (e.g., "regtest", "testnet", or "mainnet").
	var btcParams *chaincfg.Params
	switch viper.GetString("BTC_CHAIN_CONFIG") {
	case "testnet":
		btcParams = &chaincfg.TestNet3Params
	case "mainnet":
		btcParams = &chaincfg.MainNetParams
	case "regtest":
		btcParams = &chaincfg.RegressionNetParams
	default:
		// default to regtest
		btcParams = &chaincfg.RegressionNetParams
	}

	// If your Schnorr signer is created separately, load or initialize it here.
	// For this example, we assume you have a local schnorr signer that does that.
	schnorrSigner, err := multisig.NewLocalSchnorrSigner([]byte(viper.GetString("BTC_CORE_ACCOUNT_PRIV")))
	if err != nil {
		fmt.Printf("Error creating schnorr signer: %s", err)
		return nil
	}

	// *** end of preparing objects ***

	return &cmd.BridgeServerConfig{
		// eth side
		EthRpcUrl:          viper.GetString("ETH_RPC_URL"),
		EthCoreAccountPriv: viper.GetString("ETH_CORE_ACCOUNT_PRIV"),
		MSchnorrSigner:     schnorrSigner,
		// state side
		DbFilePath: viper.GetString("DB_FILE_PATH"),
		// btc side
		BtcRpcServer:       viper.GetString("BTC_RPC_SERVER"),
		BtcRpcPort:         viper.GetString("BTC_RPC_PORT"),
		BtcRpcUsername:     viper.GetString("BTC_RPC_USERNAME"),
		BtcRpcPwd:          viper.GetString("BTC_RPC_PWD"),
		BtcChainConfig:     btcParams,
		BtcCoreAccountPriv: viper.GetString("BTC_CORE_ACCOUNT_PRIV"),
		BtcCoreAccountAddr: viper.GetString("BTC_CORE_ACCOUNT_ADDR"),
		// Http side
		HttpIp:   viper.GetString("HTTP_IP"),
		HttpPort: viper.GetString("HTTP_PORT"),
	}
}
