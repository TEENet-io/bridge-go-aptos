package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"encoding/hex"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/crypto"
)

// Check APT balance
func checkAPTBalance(ctx context.Context, client *aptos.Client, addressStr string) (*big.Int, error) {
	// Convert string address to AccountAddress type
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(addressStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse address: %v", err)
	}

	balance, err := client.AccountAPTBalance(address)
	if err != nil {
		return nil, fmt.Errorf("Failed to get APT balance: %v", err)
	}
	
	// Convert to big.Int
	balanceBigInt := new(big.Int).SetUint64(balance)
	
	return balanceBigInt, nil
}

// Send APT
func sendAPT(ctx context.Context, client *aptos.Client, senderAccount *aptos.Account, recipientAddressStr string, amount *big.Int) (string, error) {
	// Convert recipient address string to AccountAddress type
	recipientAddress := aptos.AccountAddress{}
	err := recipientAddress.ParseStringRelaxed(recipientAddressStr)
	if err != nil {
		return "", fmt.Errorf("Failed to parse recipient address: %v", err)
	}

	// Create transfer payload
	payload, err := aptos.CoinTransferPayload(nil, recipientAddress, amount.Uint64())
	if err != nil {
		return "", fmt.Errorf("Failed to create transfer payload: %v", err)
	}

	// Build, sign and submit transaction
	resp, err := client.BuildSignAndSubmitTransaction(senderAccount, aptos.TransactionPayload{Payload: payload})
	if err != nil {
		return "", fmt.Errorf("Failed to build, sign and submit transaction: %v", err)
	}

	// Wait for transaction confirmation
	_, err = client.WaitForTransaction(resp.Hash)
	if err != nil {
		return "", fmt.Errorf("Failed to wait for transaction confirmation: %v", err)
	}
	// Verify transaction success
	txnInfo, err := client.TransactionByHash(resp.Hash)
	if err != nil {
		return "", fmt.Errorf("Failed to get transaction info: %v", err)
	}
	// Check transaction status
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("Failed to parse user transaction info: %v", err)
	}
	
	// fmt.Printf("交易详情:\n状态: %v\nVM状态: %s\n哈希: %s\n版本: %d\n", 
	// 	userTxn.Success, userTxn.Hash, userTxn.Version)
	if userTxn.Success {
		return resp.Hash, nil
	} else {
		return "", fmt.Errorf("Transaction execution failed: %s", userTxn.VmStatus)
	}
}

// Create Aptos account
func createAptosAccount(ctx context.Context, client *aptos.Client) (*aptos.Account, error) {
	// Generate new Ed25519 private key
	privateKey, err := crypto.GenerateEd25519PrivateKey()
	if err != nil {
		return nil, fmt.Errorf("Failed to generate Ed25519 private key: %v", err)
	}
	
	// Create account from private key
	account, err := aptos.NewAccountFromSigner(privateKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to create account from private key: %v", err)
	}
	
	fmt.Printf("Created Aptos account address: %s\n", account.Address.String())
	fmt.Printf("Created Aptos account private key: %x\n", privateKey.Bytes())
	
	return account, nil
}

// Get APT from faucet
func getFaucetAPT(ctx context.Context, client *aptos.Client, addressStr string) error {
	// Convert string address to AccountAddress type
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(addressStr)
	if err != nil {
		return fmt.Errorf("Failed to parse address: %v", err)
	}
	
	// Use client's Fund method to get APT from faucet sample from ./aptos-go-sdk/examples/multi_agent/main.go
	const fundAmount = 100_000_000 // Same amount as in the example
	err = client.Fund(address, fundAmount)
	if err != nil {
		return fmt.Errorf("Failed to get APT from faucet: %v", err)
	}

	// Print transaction hash
	fmt.Printf("Successfully got %d APT to address: %s\n", fundAmount, addressStr)
	return nil
}

// Print APT balance
func printAPTBalance(ctx context.Context, client *aptos.Client, addressStr string) error {
	balance, err := checkAPTBalance(ctx, client, addressStr)
	if err != nil {
		return err
	}
	
	fmt.Printf("APT balance for address %s: %s\n", addressStr, balance.String())
	return nil
}




// Create Aptos client
func createClient() (*aptos.Client, error) {
	// Get network configuration from environment variable, default to devnet
	networkConfig := aptos.DevnetConfig // TODO mainnet

	// Create client
	client, err := aptos.NewClient(networkConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}
	return client, nil
}

// Create account from private key
func createAccountFromPrivateKey(privateKeyHex string) (*aptos.Account, error) {
	// Decode private key
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key: %v", err)
	}

	// Create Ed25519 private key
	key := crypto.Ed25519PrivateKey{}
	err = key.FromBytes(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Ed25519 private key: %v", err)
	}

	// Create account from signer
	account, err := aptos.NewAccountFromSigner(&key)
	if err != nil {
		return nil, fmt.Errorf("Failed to create account from private key: %v", err)
	}

	return account, nil
}
// APTOS build and submit transaction
func buildAndSubmitTransaction(
	ctx context.Context,
	client *aptos.Client,
	account *aptos.Account,
	function string,
	typeArgs []string,
	args []interface{},
) (string, error) {
	// Build module ID and function name
	parts := strings.Split(function, "::")
	if len(parts) != 3 {
		return "", fmt.Errorf("Invalid function format, should be 'address::module::function'")
	}
	
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(parts[0])
	if err != nil {
		return "", fmt.Errorf("Failed to parse address: %v", err)
	}
	
	moduleId := aptos.ModuleId{
		Address: address,
		Name:    parts[1],
	}
	
	var typeTags []aptos.TypeTag
	for _, typeArg := range typeArgs {
		typeTag, err := aptos.ParseTypeTag(typeArg)
		if err != nil {
			return "", fmt.Errorf("Failed to parse type argument: %v", err)
		}
		typeTags = append(typeTags, *typeTag)
	}
	
	var argsBytes [][]byte
	for _, arg := range args {
		var argBytes []byte
		var err error
		
		switch v := arg.(type) {
		case string:
			if strings.HasPrefix(v, "0x") {
				// Handle address
				addr := aptos.AccountAddress{}
				err = addr.ParseStringRelaxed(v)
				if err != nil {
					return "", fmt.Errorf("Failed to parse address parameter: %v", err)
				}
				argBytes, err = bcs.Serialize(&addr)
			} else {
				// Handle normal string - not directly using bcs.Serialize
				// According to the official example, strings need special handling
				argBytes = []byte(v)
			}
		case uint64:
			argBytes, err = bcs.SerializeU64(v)
		case bool:
			argBytes, err = bcs.SerializeBool(v)
		default:
			return "", fmt.Errorf("Unsupported parameter type: %T", arg)
		}
		
		if err != nil {
			return "", fmt.Errorf("Failed to serialize parameter: %v", err)
		}
		argsBytes = append(argsBytes, argBytes)
	}
	
	// Build transaction payload
	payload := aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module:   moduleId,
			Function: parts[2],
			ArgTypes: typeTags,
			Args:     argsBytes,
		},
	}

	// Build, sign and submit transaction
	resp, err := client.BuildSignAndSubmitTransaction(account, payload)
	if err != nil {
		return "", fmt.Errorf("Failed to build, sign and submit transaction: %v", err)
	}

	// Wait for transaction confirmation
	_, err = client.WaitForTransaction(resp.Hash)
	if err != nil {
		return "", fmt.Errorf("Failed to wait for transaction confirmation: %v", err)
	}

	return resp.Hash, nil
}
