package main

import (
	"fmt"
	"os"

	"github.com/TEENet-io/bridge-go/logconfig"
	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/btcsuite/btcd/chaincfg"

	"github.com/TEENet-io/bridge-go/cmd"
	logger "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	ENV_CONFIG_FILE_PATH = "BRIDGE_CONFIG"
)

func main() {
	// Set overall config level to Debug
	logconfig.ConfigInfoLogger()

	// Tool to read environment variables
	viper.AutomaticEnv()

	// Accessing an environment variable of configuration file location.
	_config_file := viper.GetString(ENV_CONFIG_FILE_PATH)
	fmt.Printf("Bridge server configuration file = %s\n", _config_file)

	// See if file exists
	if !cmd.FileExists(_config_file) {
		fmt.Printf("Failed to find bridge server configuration file: %s\n", _config_file)
		return
	}
	// Read from config file.
	success := initializeViper(_config_file)
	if !success {
		return
	}
	fmt.Printf("Successfully initialized viper\n")
	// Make the configuration
	bsc := PrepareBridgeServerConfig()
	if bsc == nil {
		fmt.Printf("Error loading bridge server configuration\n")
		return
	}
	fmt.Printf("Successfully prepared bridge server configuration\n")

	logger.Info("Starting bridge server... press Ctrl+C to kill the server")
	// Start server and block.
	cmd.StartBridgeServerAndWait(bsc)
	fmt.Printf("Bridge server started\n")
}

func initializeViper(filePath string) bool {
	viper.SetConfigFile(filePath)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading configuration file, %s", err)
		return false
	}
	return true
}

// Mutisign (remote signer connector)
func setupRemoteSignerConnector(connConfig multisig_client.ConnectorConfig) (*multisig_client.Connector, error) {
	if _, err := os.Stat(connConfig.Cert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.Key); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.CaCert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.ServerCACert); os.IsNotExist(err) {
		return nil, err
	}
	c, err := multisig_client.NewConnector(&connConfig)
	return c, err
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
	var schnorrSigner multisig_client.SchnorrSigner
	var err error
	if viper.GetBool("USE_REMOTE_SIGNER") {
		// Multisign configuration (remote signer)
		var remoteSignerConfig = multisig_client.ConnectorConfig{
			UserID:        viper.GetInt("REMOTE_SIGNER_USER_ID"),
			Name:          viper.GetString("REMOTE_SIGNER_NAME"),
			Cert:          viper.GetString("REMOTE_SIGNER_CERT"),
			Key:           viper.GetString("REMOTE_SIGNER_KEY"),
			CaCert:        viper.GetString("REMOTE_SIGNER_CA_CERT"),
			ServerAddress: viper.GetString("REMOTE_SIGNER_SERVER"),
			ServerCACert:  viper.GetString("REMOTE_SIGNER_SERVER_CA_CERT"),
		}
		connector, err := setupRemoteSignerConnector(remoteSignerConfig)
		if err != nil {
			logger.Fatalf("failed to create grpc connector: %v", err)
			return nil
		}
		schnorrSigner = multisig_client.NewRemoteSchnorrSigner(connector)
		logger.WithFields(logger.Fields{
			"remote_signer_server": viper.GetString("REMOTE_SIGNER_SERVER"),
		}).Info("Using remote schnorr signer")
	} else {
		// For this example, we init a local one or a specific remote one.
		schnorrSigner, err = multisig_client.NewLocalSchnorrSigner([]byte(viper.GetString("BTC_CORE_ACCOUNT_PRIV")))
		if err != nil {
			fmt.Printf("Error creating schnorr signer: %s", err)
			return nil
		}
		logger.Info("Using local schnorr signer")
	}

	// *** end of preparing objects ***

	return &cmd.BridgeServerConfig{

		// aptos side
		AptosRpcUrl:          viper.GetString("APTOS_RPC_URL"),
		AptosCoreAccountPriv: viper.GetString("APTOS_CORE_ACCOUNT_PRIV"),
		AptosModuleAddress:   viper.GetString("APTOS_MODULE_ADDRESS"),

		// eth side
		// EthRpcUrl:          viper.GetString("ETH_RPC_URL"),
		// EthCoreAccountPriv: viper.GetString("ETH_CORE_ACCOUNT_PRIV"),
		// EthRetroScanBlk:    viper.GetInt64("ETH_RETRO_SCAN_BLK"),
		MSchnorrSigner: schnorrSigner,
		// state side
		DbFilePath: viper.GetString("DB_FILE_PATH"),
		// btc side
		BtcRpcServer:       viper.GetString("BTC_RPC_SERVER"),
		BtcRpcPort:         viper.GetString("BTC_RPC_PORT"),
		BtcRpcUsername:     viper.GetString("BTC_RPC_USERNAME"),
		BtcRpcPwd:          viper.GetString("BTC_RPC_PWD"),
		BtcChainConfig:     btcParams,
		BtcStartBlk:        viper.GetInt64("BTC_START_BLK"),
		BtcCoreAccountPriv: viper.GetString("BTC_CORE_ACCOUNT_PRIV"),
		BtcCoreAccountAddr: viper.GetString("BTC_CORE_ACCOUNT_ADDR"),
		// Http side
		HttpIp:   viper.GetString("HTTP_IP"),
		HttpPort: viper.GetString("HTTP_PORT"),
		// predefined smart contract address (if any)
		PredefinedBridgeContractAddr: viper.GetString("PREDEFINED_BRIDGE_ADDRESS"),
		PredefinedTwbtcContractAddr:  viper.GetString("PREDEFINED_TWBTC_ADDRESS"),
	}
}
