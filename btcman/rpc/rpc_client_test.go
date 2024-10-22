package rpc

import (
	"os"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"

	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
)

var server string
var port string
var username string
var password string

// Initial setup for server
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

func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}

func TestListUtxosLegacy(t *testing.T) {
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	r, err := NewRpcClient(server, port, username, password)
	if err != nil {
		t.Fatal("Cannot create PpcClient with given credentials")
	}
	defer r.Close()

	priv_key_str := "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"
	b_wallet, err := assembler.NewBasicSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatal("Cannot create BasicWallet")
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatal("Cannot create Legacy Wallet")
	}
	t.Logf("Address: %s", wallet.P2PKH.EncodeAddress()) // n35HM2LhWXLFAB7SjjNZVm4swfnDpLiKex

	utxos, err := r.GetUtxoList((wallet.P2PKH))
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet.P2PKH.EncodeAddress(), err)
	}
	for idx, item := range utxos {
		t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	}
}

// Legacy wallet transfers some bitcoins out to an address
func TestLegacySignerTransfer(t *testing.T) {
	// Set up rpc client
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	r, err := NewRpcClient(server, port, username, password)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	}
	defer r.Close()
	// Create a legacy wallet
	priv_key_str := "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"
	b_wallet, err := assembler.NewBasicSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create wallet from private key %s", priv_key_str)
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatalf("Cannot create legacy wallet")
	}

	// Query for UTXOs that we can spend
	t.Logf("Address: %s", wallet.P2PKH.EncodeAddress()) // mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT

	utxos, err := r.GetUtxoList((wallet.P2PKH))
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet.P2PKH.EncodeAddress(), err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", wallet.P2PKH.EncodeAddress())
	}
	t.Logf("utxo found: %d", len(utxos))
	for idx, item := range utxos {
		t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	}

	// Make up a transfer to another address
	var dst_amount = int64(5 * 1e8)
	// dst_addr := "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn" // let's try a legacy receiver.
	dst_addr := "bcrt1qa3ma47jt8mdqq699vv2f0f0ahpp66f2tj0pa0f" // let's try a segwit receiver.
	change_addr := wallet.P2PKH.EncodeAddress()                // to wallet itself
	var fee_amount = int64(0.1 * 1e8)

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), dst_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	t.Logf("utxo selected: %d", len(selected_utxos))

	// Make up the Tx to be sent
	tx, err := wallet.MakeTransferOutTx(dst_addr, dst_amount, change_addr, fee_amount, selected_utxos)
	if err != nil {
		t.Fatalf("cannot create transfer Tx %v", err)
	}

	t.Logf("tx.TxIn %d", len(tx.TxIn))
	t.Logf("tx.TxOut %d", len(tx.TxOut))

	// Send via RPC
	txHash, err := r.SendRawTx(tx)
	if err != nil {
		t.Fatalf("send raw Tx error, %v", err)
	}

	t.Logf("Transaction sent, txHash is %s", txHash.String())
}

func TestLegacySignerMakeBridgeDeposit(t *testing.T) {
	// Legacy wallet make a deposit on our bridge
	// Set up rpc client
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	r, err := NewRpcClient(server, port, username, password)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	}
	defer r.Close()

	// Create a legacy wallet
	priv_key_str := "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"
	b_wallet, err := assembler.NewBasicSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create wallet from private key %s", priv_key_str)
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatalf("Cannot create legacy wallet")
	}

	// Query for UTXOs that we can spend
	t.Logf("Address: %s", wallet.P2PKH.EncodeAddress()) // mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT

	utxos, err := r.GetUtxoList((wallet.P2PKH))
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet.P2PKH.EncodeAddress(), err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", wallet.P2PKH.EncodeAddress())
	}
	t.Logf("utxo found: %d", len(utxos))
	for idx, item := range utxos {
		t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	}

	// Make up a fake bridge deposit
	var btc_bridge_amount = int64(5 * 1e8)                     // deposit 5 btc
	btc_bridge_address := "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn" // let's try a legacy receiver as bridge receiver.
	btc_change_addr := wallet.P2PKH.EncodeAddress()            // send back to wallet
	var fee_amount = int64(0.1 * 1e8)

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), btc_bridge_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	t.Logf("utxo selected: %d", len(selected_utxos))

	// Make the Tx
	tx, err := wallet.MakeBridgeDepositTx(
		selected_utxos,
		btc_bridge_address,
		btc_bridge_amount,
		fee_amount,
		btc_change_addr,
		"0x8ddF05F9A5c488b4973897E278B58895bF87Cb24", // random pick up from etherscan.io
		1, // eth mainnet is 1
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

	t.Logf("Transaction sent, txHash is %s", txHash.String())
}
