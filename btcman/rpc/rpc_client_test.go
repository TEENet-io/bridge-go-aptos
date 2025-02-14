package rpc

import (
	"os"
	"testing"

	"github.com/TEENet-io/bridge-go/btcman/assembler"
	btcutils "github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/btcsuite/btcd/chaincfg"
)

const (
	MAX_BLOCKS  = 107 // Generate > 100 blocks to get recognized balance on bitcoin core.
	MIN_BLOCKS  = 1   // Minimum step to generate blocks
	SAFE_BLOCKS = 6   // Minimum confirm threshold to consider Tx is finalized.

	SEND_SATOSHI    = 0.1 * 1e8   // 0.1 btc
	FEE_SATOSHI     = 0.001 * 1e8 // 0.001 btc
	DEPOSIT_SATOSHI = 0.2 * 1e8   // 0.2 btc

	TEST_EVM_RECEIVER = "0x8ddF05F9A5c488b4973897E278B58895bF87Cb24" // random pick up from etherscan.io
	TEST_EVM_ID       = 1                                            // eth mainnet

	// This wallet holds a lot of money.
	// Also the coinbase receiver (block mines and reward goes to this address)
	p1_legacy_priv_key_str = "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"
	p1_legacy_addr_str     = "mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT"

	// Represents a user's wallet
	p2_legacy_priv_key_str = "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"
	p2_legacy_addr_str     = "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"

	// Represents a bridge wallet
	p3_legacy_priv_key_str = "cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP"
	p3_legacy_addr_str     = "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih"
)

var (
	server   string
	port     string
	username string
	password string
)

// Initial setup for bitcoin rpc server
func setup() bool {
	server = os.Getenv("SERVER")
	port = os.Getenv("PORT")
	username = os.Getenv("USER")
	password = os.Getenv("PASS")
	if server == "" || port == "" || username == "" || password == "" {
		return false
	} else {
		return true
	}
}

func setupClient(t *testing.T) (*RpcClient, error) {
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	_config := RpcClientConfig{
		ServerAddr: server,
		Port:       port,
		Username:   username,
		Pwd:        password,
	}
	r, err := NewRpcClient(&_config)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	}
	return r, err
}

// Convert from [] to []*
func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}

func TestBalance(t *testing.T) {
	// Set up RPC
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	// Create a signer
	b_wallet, err := assembler.NewBasicSigner(p1_legacy_priv_key_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatal("cannot create BasicWallet")
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatal("cannot create Legacy Wallet")
	}

	t.Logf("Address: %s", wallet.P2PKH.EncodeAddress())

	balance, err := r.GetBalance(wallet.P2PKH, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance, error %v", err)
	}
	t.Logf("Balance (satoshi): %d", balance)
	t.Logf("Balance (btc): %f", btcutils.SatoshiToBtc(balance))
}

func TestListUtxos(t *testing.T) {
	// Set up RPC
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	// Set up Signer
	b_wallet, err := assembler.NewBasicSigner(p1_legacy_priv_key_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatal("cannot create BasicWallet")
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatal("cannot create Legacy Wallet")
	}

	t.Logf("Address: %s", wallet.P2PKH.EncodeAddress())

	utxos, err := r.GetUtxoList(wallet.P2PKH, 1)
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet.P2PKH.EncodeAddress(), err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", p1_legacy_addr_str)
	}
	t.Logf("UTXO(s) found: %d", len(utxos))

	// List too long to print
	// for idx, item := range utxos {
	// 	t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	// }
}

func TestGenerateBlocks(t *testing.T) {
	// Set up RPC
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	addr, _ := assembler.DecodeAddress(p1_legacy_addr_str, assembler.GetRegtestParams())

	// Generate blocks
	blockHashes, err := r.GenerateBlocks(MIN_BLOCKS, addr)
	if err != nil {
		t.Fatalf("cannot generate blocks, error %v", err)
	}
	t.Logf("Blocks generated: %d", len(blockHashes))
}

func TestImportPrivateKey(t *testing.T) {
	// Set up RPC
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	// Import private key
	wif, err := assembler.DecodeWIF(p1_legacy_priv_key_str)
	if err != nil {
		t.Fatalf("cannot decode private key, error %v", err)
	}

	err = r.ImportPrivateKey(wif, "p1_legacy_priv_key")
	if err != nil {
		t.Fatalf("cannot import private key, error %v", err)
	}
	t.Logf("Private key imported")
}

// 1) p1 (legacy wallet) sends btc to p2 (legacy wallet)
// 2) mine the blocks (rewards to p1).
// 3) check the increased balance of p2
func TestLegacySignerTransfer(t *testing.T) {
	// Set up RPC
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	// Create a sender (p1)
	b_wallet, err := assembler.NewBasicSigner(p1_legacy_priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", p1_legacy_addr_str)
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatalf("cannot create legacy wallet")
	}
	t.Logf("Sender: %s", wallet.P2PKH.EncodeAddress())

	// Query for UTXOs that we can spend
	utxos, err := r.GetUtxoList(wallet.P2PKH, 1)
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet.P2PKH.EncodeAddress(), err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", wallet.P2PKH.EncodeAddress())
	}
	t.Logf("utxo found: %d", len(utxos))

	// List is too long to print
	// for idx, item := range utxos {
	// 	t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	// }

	// Make a transfer to another address
	dst_amount := int64(SEND_SATOSHI)
	dst_addr := p2_legacy_addr_str // receiver is p2

	t.Logf("Receiver: %s", dst_addr)

	fee_amount := int64(FEE_SATOSHI)

	// change = total (utxo) - send - fee
	change_addr := wallet.P2PKH.EncodeAddress() // to wallet itself

	// 1) Check balance of receiver
	p2_addr, err := assembler.DecodeAddress(p2_legacy_addr_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot decode address %s, error %v", p2_legacy_addr_str, err)
	}

	p2_balance_1, err := r.GetBalance(p2_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p2_legacy_addr_str, err)
	}

	t.Logf("Receiver balance (satoshi): %d", p2_balance_1)

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), dst_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	t.Logf("utxo selected: %d", len(selected_utxos))

	// Make the Tx
	tx, err := wallet.MakeTransferOutTx(dst_addr, dst_amount, change_addr, fee_amount, selected_utxos)
	if err != nil {
		t.Fatalf("cannot create transfer Tx %v", err)
	}

	t.Logf("tx.TxIn %d", len(tx.TxIn))
	t.Logf("tx.TxOut %d", len(tx.TxOut))

	// Send Tx via RPC
	txHash, err := r.SendRawTx(tx)
	if err != nil {
		t.Fatalf("send raw Tx error, %v", err)
	}

	t.Logf("Transaction sent, txHash is %s", txHash.String())

	// Generate enough blocks
	r.GenerateBlocks(MAX_BLOCKS, wallet.P2PKH)

	// 2) Check the balance of receiver again
	p2_balance_2, err := r.GetBalance(p2_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p2_legacy_addr_str, err)
	}

	t.Logf("Receiver balance (satoshi): %d", p2_balance_2)

	// If the balance is increased, the transfer is successful
	if p2_balance_2 > p2_balance_1 {
		t.Logf("Transfer successful")
	} else {
		t.Fatalf("Transfer failed")
	}
}

// 1) User (p2, legacy) make a bridge deposit to bridge wallet (p3, legacy)
// 2) mine the blocks (rewards to p1).
// 3) check the increased balance of (p3) bridge wallet
func TestLegacySignerBridgeDeposit(t *testing.T) {
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	} else {
		defer r.Close()
	}

	// Create a sender (p2)
	b_wallet, err := assembler.NewBasicSigner(p2_legacy_priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", p2_legacy_priv_key_str)
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatalf("cannot create legacy wallet")
	}

	wallet_addr_str := wallet.P2PKH.EncodeAddress()
	t.Logf("Sender: %s", wallet_addr_str)

	// Query for UTXOs of (p2)
	utxos, err := r.GetUtxoList(wallet.P2PKH, 1)
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet_addr_str, err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", wallet_addr_str)
	}
	t.Logf("utxo found: %d", len(utxos))

	// List is too long to print
	// for idx, item := range utxos {
	// 	t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	// }

	// Make up a fake bridge deposit
	deposit_amount := int64(DEPOSIT_SATOSHI) // deposit ? btc
	bridge_address := p3_legacy_addr_str     // bridge wallet
	change_addr := wallet_addr_str           // send back to p2 wallet
	fee_amount := int64(FEE_SATOSHI)         // fee

	// 1) Check balance of bridge
	p3_addr, err := assembler.DecodeAddress(p3_legacy_addr_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot decode address %s, error %v", p3_legacy_addr_str, err)
	}

	p3_balance_1, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p3_legacy_addr_str, err)
	}

	t.Logf("Bridge balance (satoshi): %d", p3_balance_1)

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), deposit_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	t.Logf("utxo selected: %d", len(selected_utxos))

	// Make the Tx
	tx, err := wallet.MakeBridgeDepositTx(
		selected_utxos,
		bridge_address,
		deposit_amount,
		fee_amount,
		change_addr,
		TEST_EVM_RECEIVER,
		TEST_EVM_ID,
	)
	if err != nil {
		t.Fatalf("cannot create Tx %v", err)
	}

	t.Logf("tx.TxIn %d", len(tx.TxIn))
	t.Logf("tx.TxOut %d", len(tx.TxOut))

	// Send via RPC
	txHash, err := r.SendRawTx(tx)
	if err != nil {
		t.Fatalf("send raw Tx error, %v", err)
	}

	t.Logf("transaction sent, txHash is %s", txHash.String())

	// Generate enough blocks
	p1_addr, _ := assembler.DecodeAddress(p1_legacy_addr_str, assembler.GetRegtestParams())
	r.GenerateBlocks(MAX_BLOCKS, p1_addr)

	// Check the p3 balance again
	p3_balance_2, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p3_legacy_addr_str, err)
	}

	t.Logf("Bridge balance (satoshi): %d", p3_balance_2)

	// If the balance is increased, the transfer is successful
	if p3_balance_2 > p3_balance_1 {
		t.Logf("Transfer successful")
	} else {
		t.Fatalf("Transfer failed")
	}
}
