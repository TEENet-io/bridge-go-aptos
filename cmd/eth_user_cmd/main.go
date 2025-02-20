package main

import (
	"bufio"
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/TEENet-io/bridge-go/cmd"
	"github.com/spf13/viper"
)

const (
	ENV_CONFIG_FILE_PATH = "ETH_USER_CONFIG"
)

func main() {
	// Set overall config level to Debug
	// logconfig.ConfigDebugLogger()

	// Tool to read environment variables
	viper.AutomaticEnv()

	// Accessing an environment variable of configuration file location.
	_config_file := viper.GetString(ENV_CONFIG_FILE_PATH)
	fmt.Printf("ETH user configuration file = %s\n", _config_file)

	// See if file exists
	if !cmd.FileExists(_config_file) {
		fmt.Printf("ETH user configuration file not found: %s\n", _config_file)
		return
	}

	// Read from config file.
	success := initializeViper(_config_file)
	if !success {
		return
	}

	// Prepre the BTC user configuration
	euc := PrepareEthUserConfig()
	eu, err := cmd.NewEthUser(euc)
	if err != nil {
		fmt.Printf("Error creating BTC user: %s\n", err)
		return
	}

	fmt.Println(strings.Repeat("=", 30))
	fmt.Println("Welcome to bridge ETH user command line tool.")
	fmt.Printf("Connected to: %s\n", euc.EthRpcUrl)
	fmt.Printf("ChainId: %d\n", eu.ChainId.Int64())
	fmt.Printf("Your ETH address: %s\n", eu.GetAddress())
	fmt.Printf("TWBTC contract address: %s\n", euc.EthTwbtcAddress)
	fmt.Printf("Bridge contract address: %s\n", euc.EthBridgeAddress)

	// *** user interactive program ***

	// Create a cancelable context and signal handler for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handler to catch Ctrl-C.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		_captured := <-sig
		fmt.Printf("\nReceived interrupt signal, shutting down... %v\n", _captured)
		cancel()
		os.Exit(0)
	}()

	// gather user inputs
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// Check if context is done (just in case)
		select {
		case <-ctx.Done():
			eu.Close() // release btcUser resources.
			return
		default:
		}

		// Print options
		fmt.Println("What to do:")
		fmt.Println("1) View balance")
		fmt.Println("2) Initiate redeem")
		fmt.Print("Type option and press Enter: ")

		// Wait for input.
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		// Process user input.
		switch input {
		case "1":
			_balance, err1 := eu.GetBalance()
			_twbtc, err2 := eu.GetTwbtcBalance()
			if err1 != nil || err2 != nil {
				fmt.Printf("Error getting balance: %s, %s\n", err1, err2)
			} else {
				fmt.Printf("Your ETH address: %s\n", eu.GetAddress())
				fmt.Printf("Your balance: %d ETH (wei), %d TWBTC (wei)\n", _balance, _twbtc)
			}
		case "2":
			fmt.Println("action 2!")
			tx_id, err := sendRedeem(eu)
			if err != nil {
				fmt.Printf("Error sending redeem: %s\n", err)
			} else {
				fmt.Printf("Redeem request sent, tx_id: %s\n", tx_id)
			}
		default:
			fmt.Println("Unknown option, try again.")
		}
		fmt.Println()
	}
}

func initializeViper(filePath string) bool {
	viper.SetConfigFile(filePath)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading configuration file, %s", err)
		return false
	}
	return true
}

func PrepareEthUserConfig() *cmd.EthUserConfig {
	euc := &cmd.EthUserConfig{
		EthRpcUrl:          viper.GetString("ETH_RPC_URL"),
		EthCoreAccountPriv: viper.GetString("ETH_CORE_ACCOUNT_PRIV"),
		EthBridgeAddress:   viper.GetString("ETH_BRIDGE_ADDRESS"),
		EthTwbtcAddress:    viper.GetString("ETH_TWBTC_ADDRESS"),
	}
	return euc
}

func sendRedeem(eu *cmd.EthUser) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter receiver of this redeem (BTC address): ")
	scanner.Scan()
	receiverBtcAddress := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter amount to redeem (in satoshis, must > 100,000): ")
	scanner.Scan()
	amountToRedeem, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid amount: %s\n", err)
		return "", err
	}

	if amountToRedeem < 100000 {
		return "", fmt.Errorf("amount must be greater than 100,000 otherwise considered as dust attack on BTC")
	}

	fmt.Printf("Sending redeem request to bridge... receiver: %s, amount: %d\n", receiverBtcAddress, amountToRedeem)

	_twbtc_balance, err := eu.GetTwbtcBalance()
	if err != nil {
		fmt.Printf("Error getting TWBTC balance: %s\n", err)
		return "", err
	}
	if _twbtc_balance.Cmp(big.NewInt(int64(amountToRedeem))) < 0 {
		fmt.Printf("Insufficient TWBTC balance: %d\n", _twbtc_balance)
		return "", fmt.Errorf("insufficient TWBTC balance")
	}

	_redeem_tx_hash, err := eu.InitiateRedeem(big.NewInt(int64(amountToRedeem)), receiverBtcAddress)
	if err != nil {
		fmt.Printf("Error when sending redeem request: %s\n", err)
		return "", err
	}

	return _redeem_tx_hash.Hex(), nil
}
