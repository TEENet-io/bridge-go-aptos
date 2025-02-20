package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/TEENet-io/bridge-go/cmd"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/spf13/viper"
)

const (
	ENV_CONFIG_FILE_PATH = "BTC_USER_CONFIG"
)

func main() {
	// Set overall config level to Debug
	// logconfig.ConfigDebugLogger()

	// Tool to read environment variables
	viper.AutomaticEnv()

	// Accessing an environment variable of configuration file location.
	_config_file := viper.GetString(ENV_CONFIG_FILE_PATH)
	fmt.Printf("BTC user configuration file = %s\n", _config_file)

	// See if file exists
	if !cmd.FileExists(_config_file) {
		fmt.Printf("BTC user configuration file not found: %s\n", _config_file)
		return
	}

	// Read from config file.
	success := initializeViper(_config_file)
	if !success {
		return
	}

	// Prepre the BTC user configuration
	buc := PrepareBtcUserConfig()
	if buc == nil {
		fmt.Printf("Error prepare BTC user configuration\n")
		return
	}

	bu, err := cmd.NewBtcUser(buc, viper.GetBool("REGISTER_USER_ON_CHAIN"))
	if err != nil {
		fmt.Printf("Error creating BTC user: %s\n", err)
		return
	}

	fmt.Println(strings.Repeat("=", 30))
	fmt.Println("Welcome to bridge BTC user command line tool.")
	fmt.Printf("Your BTC address: %s\n", bu.MyUserConfig.BtcCoreAccountAddr)

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
			bu.Close() // release btcUser resources.
			return
		default:
		}

		// Print options
		fmt.Println("What to do:")
		fmt.Println("1) View balance")
		fmt.Println("2) View recent UTXOs")
		fmt.Println("3) Send deposit to bridge")
		fmt.Println("4) Tell BTC network to mine blocks (regtest only)")
		fmt.Println("5) Transfer BTC to another address")
		fmt.Print("Type option and press Enter: ")

		// Wait for input.
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		// Process user input.
		switch input {
		case "1":
			_balance, err := bu.GetBalance()
			if err != nil {
				fmt.Printf("Error getting balance: %s\n", err)
			} else {
				fmt.Printf("Your BTC address: %s\n", bu.MyUserConfig.BtcCoreAccountAddr)
				fmt.Printf("Your balance: %d\n", _balance)
			}
		case "2":
			_utxos, err := bu.GetUtxos()
			if err != nil {
				fmt.Printf("Error getting UTXOs: %s\n", err)
			} else {
				for idx, _utxo := range _utxos {
					fmt.Printf("[%d]: TxId %s, vout %d, %d satoshi\n", idx, _utxo.TxID, _utxo.Vout, _utxo.Amount)
				}
			}
			// Your action 2 code here.
		case "3":
			fmt.Println("Sending a deposit to bridge...")
			sendDeposit(bu)
		case "4":
			fmt.Println("Only use this option in local regtest mode.")
			fmt.Println("In production / public node it has no effect!")
			_blks, err := bu.MineEnoughBlocks()
			if err != nil {
				fmt.Printf("Error mining blocks: %s\n", err)
			} else {
				fmt.Printf("Mined %d blocks\n", len(_blks))
			}
		case "5":
			fmt.Println("Sending some btc to another address...")
			sendTransfer(bu)
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

func PrepareBtcUserConfig() *cmd.BtcUserConfig {
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

	return &cmd.BtcUserConfig{
		BtcRpcServer:       viper.GetString("BTC_RPC_SERVER"),
		BtcRpcPort:         viper.GetString("BTC_RPC_PORT"),
		BtcRpcUsername:     viper.GetString("BTC_RPC_USERNAME"),
		BtcRpcPwd:          viper.GetString("BTC_RPC_PWD"),
		BtcChainConfig:     btcParams,
		BtcCoreAccountPriv: viper.GetString("BTC_CORE_ACCOUNT_PRIV"),
		BtcCoreAccountAddr: viper.GetString("BTC_CORE_ACCOUNT_ADDR"),
	}
}

func sendDeposit(bu *cmd.BtcUser) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter deposit address of bridge (BTC): ")
	scanner.Scan()
	bridgeBtcWalletAddress := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter amount to send (in satoshis): ")
	scanner.Scan()
	amountToSend, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid amount: %s\n", err)
		return "", err
	}

	if amountToSend < 100000 {
		return "", fmt.Errorf("amount must be greater than 100,000 otherwise considered as dust attack on BTC")
	}

	fmt.Print("Enter Tx fee amount (in satoshis, recommend minimum 100000 for regtest): ")
	scanner.Scan()
	feeAmount, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid fee amount: %s\n", err)
		return "", err
	}

	fmt.Print("Enter the eth address of receiver on Ethereum: ")
	scanner.Scan()
	ethReceiverAddress := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter Ethereum network ID (number): ")
	scanner.Scan()
	ethNetworkID, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid Ethereum network ID: %s\n", err)
		return "", err
	}

	fmt.Printf("Sending %d satoshis to %s with a fee of %d satoshis. Receiver on Ethereum: %s, Network ID: %d\n",
		amountToSend, bridgeBtcWalletAddress, feeAmount, ethReceiverAddress, ethNetworkID)

	// check if user's balance is enough
	balance, err := bu.GetBalance()
	if err != nil {
		fmt.Printf("Error getting balance: %s\n", err)
		return "", err
	}
	if balance < int64(amountToSend+feeAmount) {
		fmt.Printf("Not enough balance: have %d, need %d\n", balance, amountToSend+feeAmount)
	}
	// Here you would call the function to send the deposit using the collected inputs.
	btcTxId, err := bu.DepositToBridge(int64(amountToSend), int64(feeAmount), bridgeBtcWalletAddress, ethReceiverAddress, ethNetworkID)
	if err != nil {
		fmt.Printf("Error sending deposit: %s\n", err)
		return "", err
	}
	return btcTxId, nil
}

func sendTransfer(bu *cmd.BtcUser) (string, error) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter receiver address: ")
	scanner.Scan()
	receiverBtcWalletAddress := strings.TrimSpace(scanner.Text())

	fmt.Print("Enter amount to send (in satoshis): ")
	scanner.Scan()
	amountToSend, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid amount: %s\n", err)
		return "", err
	}

	if amountToSend < 100000 {
		return "", fmt.Errorf("amount must be greater than 100,000 otherwise considered as dust attack on BTC")
	}

	fmt.Print("Enter Tx fee amount (in satoshis, recommend minimum 100000 for regtest): ")
	scanner.Scan()
	feeAmount, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil {
		fmt.Printf("Invalid fee amount: %s\n", err)
		return "", err
	}

	fmt.Printf("Sending %d satoshis to %s with a fee of %d satoshis.\n",
		amountToSend, receiverBtcWalletAddress, feeAmount)

	// check if user's balance is enough
	balance, err := bu.GetBalance()
	if err != nil {
		fmt.Printf("Error getting balance: %s\n", err)
		return "", err
	}
	if balance < int64(amountToSend+feeAmount) {
		fmt.Printf("Not enough balance: have %d, need %d\n", balance, amountToSend+feeAmount)
	}
	// Here you would call the function to send the deposit using the collected inputs.
	btcTxId, err := bu.TransferOut(int64(amountToSend), int64(feeAmount), receiverBtcWalletAddress)
	if err != nil {
		fmt.Printf("Error sending transfer: %s\n", err)
		return "", err
	}
	return btcTxId, nil
}
