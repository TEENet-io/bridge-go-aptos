package btcsync

// Test btc monitor for BTC deposit
// 1. Send the deposit
// 2. Monitor captures the deposit
// 3. Monitor Triggers downstream actions.

import (
	"os"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	sharedcommon "github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	CHANNEL_BUFFER_SIZE = 10

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

// RPC Server configs.
var (
	server   string
	port     string
	username string
	password string
)

// Convert from [] to []*
func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}

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

func setupClient(t *testing.T) (*rpc.RpcClient, error) {
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	r, err := rpc.NewRpcClient(server, port, username, password)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	}
	return r, err
}

func randFileName(prefix string, suffix string) string {
	return prefix + ethcommon.Hash(sharedcommon.RandBytes32()).String() + suffix
}

// call it in defer
func rmFile(name string) {
	os.Remove(name)
}

// call it once to get the db file name in this run.
func setupDBFile() string {
	db_file_name := randFileName("test_", ".db")
	return db_file_name
}

// Set up the BTC Monitor
func setupMonitor(t *testing.T, r *rpc.RpcClient, st btcaction.RedeemActionStorage, startBlock int) (*BTCMonitor, error) {

	// Create a new monitor instance
	// monitor on p3
	monitor, err := NewBTCMonitor(
		p3_legacy_addr_str,
		assembler.GetRegtestParams(),
		r,
		int64(startBlock),
		st,
	)
	if err != nil {
		t.Fatalf("cannot create monitor %v", err)
	}

	return monitor, nil
}

func setupObserverDeposit(st btcaction.DepositStorage) (*ObserverDepositAction, error) {
	return NewObserverDepositAction(st, CHANNEL_BUFFER_SIZE), nil
}

func TestDeposit(t *testing.T) {
	// Setup the rpc client (shared between monitor and deposit sender)
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create rpc client")
	}
	defer r.Close()

	// Setup the db file name
	db_file_name := setupDBFile()
	defer rmFile(db_file_name)
	t.Logf("db file name: %s", db_file_name)

	// Setup the sqlite db for manager state (mainly for redeem)
	internal_st, err := btcaction.NewSQLiteRedeemStorage(db_file_name)
	if err != nil {
		t.Fatalf("cannot create backend storage %v", err)
	}

	// Setup the btc monitor
	monitor, err := setupMonitor(t, r, internal_st, 0)
	if err != nil {
		t.Fatalf("cannot create monitor, %v", err)
	}

	// Setup the observers

	// Deposit observer
	// 1) Create deposit storage
	depo_st, err := btcaction.NewSQLiteDepositStorage(db_file_name)
	if err != nil {
		t.Fatalf("cannot create deposit storage %v", err)
	}
	// 2) Create observer
	d_observer, err := setupObserverDeposit(depo_st)
	if err != nil {
		t.Fatalf("cannot create monitor, %v", err)
	}
	// start listening to the deposit channel
	go d_observer.GetNotifiedDeposit()

	// Register Deposit Observer to publisher
	monitor.Publisher.RegisterDepositObserver(d_observer.Ch)

	// Turn on the monitor scan loop
	go monitor.ScanLoop()

	// Send the deposit p2 -> p3
	// Create a sender (p2)
	b_wallet, err := assembler.NewBasicSigner(p2_legacy_priv_key_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", p2_legacy_priv_key_str)
	}
	wallet, err := assembler.NewLegacySigner(*b_wallet)
	if err != nil {
		t.Fatalf("cannot create legacy wallet")
	}

	wallet_addr_str := wallet.P2PKH.EncodeAddress()
	t.Logf("Deposit Sender: %s", wallet_addr_str)

	// Query for UTXOs of (p2)
	// It simulates a personal user's wallet.
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

	// Make the Deposit Tx
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

	// Check on monitor side if the deposit is captured
	// Check if the deposit is captured in the storage
	var deposits []btcaction.DepositAction
	for i := 0; i < 10; i++ {
		deposits, err = depo_st.GetDepositByTxHash(txHash.String())
		if err != nil {
			t.Fatalf("error retrieving deposit by tx hash: %v", err)
		}
		if len(deposits) > 0 {
			break
		}
		time.Sleep(3 * time.Second)
	}

	if len(deposits) > 0 {
		t.Logf("Deposit captured successfully")
	} else {
		t.Fatalf("Deposit not captured")
	}
}
