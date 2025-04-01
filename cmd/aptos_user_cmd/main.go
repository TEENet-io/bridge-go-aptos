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
	"github.com/spf13/viper"

	"github.com/aptos-labs/aptos-go-sdk"
)


// 日志信息输出函数，带颜色
func logInfo(message string) { 
	fmt.Printf("\x1b[36m%s\x1b[0m\n", message)
}

func logSuccess(message string) { 
	fmt.Printf("\x1b[32m%s\x1b[0m\n", message)
}

func logError(message string) { 
	fmt.Printf("\x1b[31m%s\x1b[0m\n", message)
}

func logWarning(message string) { 
	fmt.Printf("\x1b[33m%s\x1b[0m\n", message)
}



// 打印主菜单选项
func printMainMenu() {
	fmt.Println("\n===== APTOS TWBTC工具 =====")
	fmt.Println("1) 检查APT余额")
	fmt.Println("2) 发送APT")
	fmt.Println("3) 创建Aptos账户")
	fmt.Println("4) 水龙头获取APT")
	fmt.Println("5) 检查TWBTC余额")
	fmt.Println("6) 初始化TWBTC")
	fmt.Println("7) 初始化桥接")
	fmt.Println("8) 注册账户TWBTC")
	fmt.Println("9) 发起赎回请求")
	fmt.Println("10) 铸币")
	fmt.Println("11) 查询事件")
	fmt.Println("0) 退出程序")
	fmt.Print("请输入选项编号: ")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
		os.Exit(0)
	}()

	// Read configuration from regtest.yaml
	configFile := "./regtest.yaml"
	viper.SetConfigFile(configFile)
	
	err := viper.ReadInConfig()
	if err != nil {
		logWarning(fmt.Sprintf("WARNING: Failed to read config file %s: %v", configFile, err))
		logWarning("Some features will be unavailable. Please create the config file and restart the program.")
	}
	
	// Define config structure
	var config struct {
		PrivateKey    string
		ModuleAddress string
	}
	
	// Extract values from viper if config was loaded successfully
	if err == nil {
		config.PrivateKey = viper.GetString("APTOS_CORE_ACCOUNT_PRIV")
		config.ModuleAddress = viper.GetString("APTOS_CORE_ACCOUNT_ADDR")
	}

	// Use private key from config or fallback to environment variable
	privateKey := config.PrivateKey
	if privateKey == "" {
		logWarning("WARNING: Private key not found in config or environment. Some features will be unavailable.")
	}

	// Use module address from config or fallback to environment variable
	moduleAddress := config.ModuleAddress
	if moduleAddress == "" {
		logWarning("WARNING: Module address not found in config or environment. Some features will be unavailable.")
	}

	// Create client
	client, err := createClient()
	if err != nil {
		logError(fmt.Sprintf("Failed to create client: %v", err))
		os.Exit(1)
	}
	logSuccess("Successfully connected to Aptos network")

	var account *aptos.Account
	if privateKey != "" {
		account, err = createAccountFromPrivateKey(privateKey)
		if err != nil {
			logError(fmt.Sprintf("Failed to create account: %v", err))
			os.Exit(1)
		}
		logSuccess(fmt.Sprintf("Account loaded: %s", account.Address.String()))
	}

	scanner := bufio.NewScanner(os.Stdin)

	// Main loop
	for {
		// Check if context is done
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Print main menu
		printMainMenu()

		// Read user input
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		// Process main menu options
		switch input {
		case "0":
			fmt.Println("Exiting program...")
			return

		case "1": // Check APT balance
			fmt.Print("Enter the address to check (leave blank to use current account): ")
			scanner.Scan()
			addressStr := strings.TrimSpace(scanner.Text())
			if addressStr == "" && account != nil {
				addressStr = account.Address.String()
			} else if addressStr == "" {
				logError("No address provided and no account loaded")
				continue
			}
			err := printAPTBalance(ctx, client, addressStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to get balance: %v", err))
			}

		case "2": // Send APT
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			fmt.Print("Enter the recipient address: ")
			scanner.Scan()
			recipient := strings.TrimSpace(scanner.Text())
			
			fmt.Print("Enter the amount (APT): ")
			scanner.Scan()
			amountStr := strings.TrimSpace(scanner.Text())
			
			// Convert APT to Octas (1 APT = 10^8 Octas)
			amount, success := new(big.Float).SetString(amountStr)
			if !success {
				logError(fmt.Sprintf("Invalid amount: %s", amountStr))
				continue
			}
			
			// Convert to Octas (integer)
			amount = amount.Mul(amount, big.NewFloat(100000000))
			amountInt, _ := amount.Int(nil)
			
			txHash, err := sendAPT(ctx, client, account, recipient, amountInt)
			if err != nil {
				logError(fmt.Sprintf("Failed to send APT: %v", err))
				continue
			}
			
			logSuccess(fmt.Sprintf("Successfully sent %s APT to address %s", amountStr, recipient))
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))

		case "3": // Create Aptos account
			newAccount, err := createAptosAccount(ctx, client)
			if err != nil {
				logError(fmt.Sprintf("Failed to create account: %v", err))
				continue
			}
			logSuccess(fmt.Sprintf("Successfully created account, address: %s", newAccount.Address.String()))
			logSuccess("Please save the private key and address for future use")

		case "4": // Get APT from faucet
			fmt.Print("Enter the address to receive APT: ")
			scanner.Scan()
			addressStr := strings.TrimSpace(scanner.Text())
			if addressStr == "" {
				logError("Address cannot be empty")
				continue
			}
			
			err := getFaucetAPT(ctx, client, addressStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to get APT from faucet: %v", err))
				continue
			}
			logSuccess(fmt.Sprintf("Successfully got APT from faucet to address: %s", addressStr))
			err = printAPTBalance(ctx, client, addressStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to get balance: %v", err))
			}

		case "5": // Check TWBTC balance
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			fmt.Print("Enter the address to check (leave blank to use current account): ")
			scanner.Scan()
			addressStr := strings.TrimSpace(scanner.Text())
			if addressStr == "" && account != nil {
				addressStr = account.Address.String()
			} else if addressStr == "" {
				logError("No address provided and no account loaded")
				continue
			}
			
			address := aptos.AccountAddress{}
			err := address.ParseStringRelaxed(addressStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse address: %v", err))
				continue
			}
			
			balance, err := CheckTWBTCBalance(client, address, moduleAddress)
			if err != nil {
				logError(fmt.Sprintf("Failed to check TWBTC balance: %v", err))
				continue
			}
			
			fmt.Printf("TWBTC balance for address %s: %s Satoshis\n", addressStr, balance.String())


		case "6": // Initialize TWBTC
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			
			txHash, err := initTWBTC(client, account, moduleAddress)
			if err != nil {
				logError(fmt.Sprintf("Failed to initialize TWBTC: %v", err))
				continue
			}
			logSuccess("Successfully initialized TWBTC")
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))
			
		case "7": // Initialize bridge
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			
			fmt.Print("Enter the fee account address: ")
			scanner.Scan()
			feeAccountStr := strings.TrimSpace(scanner.Text())
			
			fmt.Print("Enter the fee (satoshi): ")
			scanner.Scan()
			feeStr := strings.TrimSpace(scanner.Text())
			
			feeAccountAddress := aptos.AccountAddress{}
			err := feeAccountAddress.ParseStringRelaxed(feeAccountStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse fee account address: %v", err))
				continue
			}
			
			fee, err := strconv.ParseUint(feeStr, 10, 64)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse fee: %v", err))
				continue
			}
			
			txHash, err := initBridge(client, account, moduleAddress, feeAccountAddress, fee)
			if err != nil {
				logError(fmt.Sprintf("Failed to initialize bridge: %v", err))
				continue
			}
			logSuccess("Successfully initialized bridge")
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))
			
		case "8": // Register account TWBTC
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			
			fmt.Print("Enter the recipient address: ")
			scanner.Scan()
			receiverStr := strings.TrimSpace(scanner.Text())
			
			receiverAddress := aptos.AccountAddress{}
			err := receiverAddress.ParseStringRelaxed(receiverStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse recipient address: %v", err))
				continue
			}
			
			txHash, err := registerTWBTC(client, account, moduleAddress, receiverAddress)
			if err != nil {
				logError(fmt.Sprintf("Failed to register TWBTC: %v", err))
				continue
			}
			logSuccess("Successfully registered TWBTC")
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))
			
		case "9": // Initiate redemption request
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			
			fmt.Print("Enter the BTC receiver address: ")
			scanner.Scan()
			receiverStr := strings.TrimSpace(scanner.Text())
			
			fmt.Print("Enter the redemption amount (satoshi): ")
			scanner.Scan()
			amountStr := strings.TrimSpace(scanner.Text())
			
			amount, err := strconv.ParseUint(amountStr, 10, 64)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse amount: %v", err))
				continue
			}
			
			txHash, err := redeemRequest(client, account, moduleAddress, receiverStr, amount)
			if err != nil {
				logError(fmt.Sprintf("Failed to redeem request: %v", err))
				continue
			}
			logSuccess(fmt.Sprintf("Successfully sent redemption request for %d Satoshis to BTC address %s", amount, receiverStr))
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))
			
		case "10": // Mint
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			if account == nil {
				logError("Need to load account to perform this operation")
				continue
			}
			
			fmt.Print("Enter the BTC transaction ID: ")
			scanner.Scan()
			btcTxId := strings.TrimSpace(scanner.Text())
			
			fmt.Print("Enter the recipient address: ")
			scanner.Scan()
			receiverStr := strings.TrimSpace(scanner.Text())
			
			fmt.Print("Enter the mint amount (satoshi): ")
			scanner.Scan()
			amountStr := strings.TrimSpace(scanner.Text())
			
			receiverAddress := aptos.AccountAddress{}
			err := receiverAddress.ParseStringRelaxed(receiverStr)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse receiver address: %v", err))
				continue
			}
			
			amount, err := strconv.ParseUint(amountStr, 10, 64)
			if err != nil {
				logError(fmt.Sprintf("Failed to parse amount: %v", err))
				continue
			}
			
			txHash, err := mintTWBTC(client, account, moduleAddress, receiverAddress, amount, btcTxId)
			if err != nil {
				logError(fmt.Sprintf("Failed to mint: %v", err))
				continue
			}
			logSuccess(fmt.Sprintf("Successfully minted %d Satoshis to address %s", amount, receiverStr))
			logSuccess(fmt.Sprintf("Transaction hash: %s", txHash))
			
		case "11": // Query events
			if moduleAddress == "" {
				logError("Module address is not set, cannot perform this operation")
				continue
			}
			
			fmt.Print("Enter the query interval time (seconds, default 60): ")
			scanner.Scan()
			intervalStr := strings.TrimSpace(scanner.Text())
			
			interval := 60
			if intervalStr != "" {
				var err error
				interval, err = strconv.Atoi(intervalStr)
				if err != nil {
					logError("Invalid time interval, using default value 60 seconds")
					interval = 60
				}
			}
			
			fmt.Println("Starting to query events, press Ctrl+C to exit...")
			err := QueryBridgeStatus(client, moduleAddress, interval)
			if err != nil {
				logError(fmt.Sprintf("Failed to query events: %v", err))
			}

		default:
			logError("Invalid option, please try again")
		}
	}
}
