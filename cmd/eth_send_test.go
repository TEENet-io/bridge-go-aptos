// This is a test to send 1 ether from one account to another account.
// It tests:
// 1. The connection to the a local Ethereum node.
// 2. The Ether transfer transaction creation.
// 3. The mining of the transaction on ETH network.
// 4. The balance of two accounts after the transaction.

package cmd_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	RPCURL = "http://localhost:8545"
	// RPCURL               = "https://eth-sepolia.public.blastapi.io"
	SENDER_PRIVATE_KEY   = "dbcec79f3490a6d5d162ca2064661b85c40c93672968bfbd906b952e38c3e8de"
	SENDER_ADDR          = "0x85b427C84731bC077BA5A365771D2b64c5250Ac8"
	RECEIVER_PRIVATE_KEY = "e751da9079ca6b4e40e03322b32180e661f1f586ca1914391c56d665ffc8ec74"
	RECEIVER_ADDR        = "0xdab133353Cff0773BAcb51d46195f01bD3D03940"
)

func TestSend(t *testing.T) {
	// Connect to the local Ethereum node
	client, err := ethclient.Dial(RPCURL)
	if err != nil {
		t.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	// Sender private key
	senderPrivateKey, err := crypto.HexToECDSA(SENDER_PRIVATE_KEY)
	if err != nil {
		t.Fatalf("Failed to load sender private key: %v", err)
	}

	// Receiver address
	receiverAddress := common.HexToAddress(RECEIVER_ADDR)

	// Get the nonce for the sender account
	senderAddress := crypto.PubkeyToAddress(senderPrivateKey.PublicKey)
	nonce, err := client.PendingNonceAt(context.Background(), senderAddress)
	if err != nil {
		t.Fatalf("Failed to get nonce: %v", err)
	}

	// Set the gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		t.Fatalf("Failed to suggest gas price: %v", err)
	}

	// Set the amount to send (in Wei)
	amount := big.NewInt(1000000000000000000 / 1000) // 0.001 ETH

	// Create the transaction
	tx := types.NewTransaction(nonce, receiverAddress, amount, uint64(21000), gasPrice, nil)

	// Sign the transaction
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		t.Fatalf("Failed to get network ID: %v", err)
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), senderPrivateKey)
	if err != nil {
		t.Fatalf("Failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		t.Fatalf("Failed to send transaction: %v", err)
	}

	t.Logf("Transaction sent: %s", signedTx.Hash().Hex())

	// Wait for the transaction to be mined
	// TODO: shall use a for{} loop and err != NOTFOUND to wait for the mining process.
	receipt, err := client.TransactionReceipt(context.Background(), signedTx.Hash())
	if err != nil {
		t.Fatalf("Failed to get transaction receipt: %v", err)
	}

	if receipt.Status != 1 {
		t.Fatalf("Transaction failed: %v", receipt)
	}
}
