package cmd_test

// Notice:
// This test uses
// 1) a local bitcoin two-node network (regtest mode) v0.21.
// 2) And a Ethereum local Geth network (regtest mode) v1.14.

// The test includes:
// 1. Set up of a real bridge server.
// 2. Connect to real Eth and Btc networks.
// 3. Test Btc Deposit (mint TWBTC on eth side)
// 4. Tset TWBTC Withdraw (burn TWBTC on eth side, get Btc back to user on btc side)

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/btcvault"
	"github.com/TEENet-io/bridge-go/cmd"
	sharedcommon "github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/logconfig"
	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/TEENet-io/bridge-go/reporter"
)

const (
	RETRY_TIMES         = 10 // retry times for checking the deposit/utxo
	CHANNEL_BUFFER_SIZE = 10

	MAX_BLOCKS = 107 // Generate > 100 blocks to get recognized balance on bitcoin core.
	MIN_BLOCKS = 1   // Minimum step to generate blocks

	// SEND_SATOSHI    = 0.1 * 1e8   // 0.1 btc
	WITHDRAW_FEE_SATOSHI = 0.0001 * 1e8 // 0.0001 btc = 10,000 satoshi
	DEPOSIT_SATOSHI      = 0.2 * 1e8    // 0.2 btc
	REDEEM_SATOSHI       = 0.1 * 1e8    // 0.1 btc (half of deposit)

	// TEST_EVM_RECEIVER = "0x8ddF05F9A5c488b4973897E278B58895bF87Cb24" // random pick up from etherscan.io
	// TEST_EVM_ID = 1 // eth mainnet

	// This btc wallet holds a lot of money.
	// Also acts the coinbase receiver (block mines and reward goes to this address)
	coinbase_legacy_addr_str     = "mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT"
	coinbase_legacy_priv_key_str = "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"

	// btc chain
	BTC_RPC_SERVER   = "127.0.0.1"
	BTC_RPC_PORT     = "19001"
	BTC_RPC_USERNAME = "admin1"
	BTC_RPC_PWD      = "123"

	// user's btc wallet
	BTC_USER_ACCOUNT_ADDR = "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"
	BTC_USER_ACCOUNT_PRIV = "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"

	// bridge's btc wallet (receive deposits from users)
	BTC_CORE_ACCOUNT_PRIV = "cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP"
	BTC_CORE_ACCOUNT_ADDR = "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih"

	// eth synchronizer config
	frequencyToCheckEthFinalizedBlock = 100 * time.Millisecond

	// eth tx manager config
	frequencyToPrepareRedeem      = 500 * time.Millisecond
	frequencyToMint               = 500 * time.Millisecond // 0.5 second
	frequencyToMonitorPendingTxs  = 500 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 10

	// eth chain
	ETH_RPC_URL = "http://localhost:8545"

	// user's eth wallet
	ETH_BRIDGE_ADDR        = "0x85b427C84731bC077BA5A365771D2b64c5250Ac8"
	ETH_BRIDGE_PRIVATE_KEY = "dbcec79f3490a6d5d162ca2064661b85c40c93672968bfbd906b952e38c3e8de"

	// bridge's eth wallet (to mint(), to redeem() and to deploy smart contracts)
	ETH_USER_ADDR        = "0xdab133353Cff0773BAcb51d46195f01bD3D03940"
	ETH_USER_PRIVATE_KEY = "e751da9079ca6b4e40e03322b32180e661f1f586ca1914391c56d665ffc8ec74"

	HTTP_IP   = "0.0.0.0"
	HTTP_PORT = "8080"
)

// Multisign configuration (remote signer)
// var remoteSignerConfig = multisig_client.ConnectorConfig{
// 	UserID:        0,
// 	Name:          "client0",
// 	Cert:          "../multisig/config/data/client0.crt",
// 	Key:           "../multisig/config/data/client0.key",
// 	CaCert:        "../multisig/config/data/client0-ca.crt",
// 	ServerAddress: "20.205.130.99:6001",
// 	ServerCACert:  "../multisig/config/data/node0-ca.crt",
// }

// Mutisign (remote signer connector)
// func setupConnector(connConfig multisig_client.ConnectorConfig) (*multisig_client.Connector, error) {
// 	if _, err := os.Stat(connConfig.Cert); os.IsNotExist(err) {
// 		return nil, err
// 	}
// 	if _, err := os.Stat(connConfig.Key); os.IsNotExist(err) {
// 		return nil, err
// 	}
// 	if _, err := os.Stat(connConfig.CaCert); os.IsNotExist(err) {
// 		return nil, err
// 	}
// 	if _, err := os.Stat(connConfig.ServerCACert); os.IsNotExist(err) {
// 		return nil, err
// 	}
// 	c, err := multisig_client.NewConnector(&connConfig)
// 	return c, err
// }

// Convert from [] to []*
func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}

// Random file name generator
func randFileName(prefix string, suffix string) string {
	return prefix + ethcommon.Hash(sharedcommon.RandBytes32()).String() + suffix
}

// call it to get the db file name.
func setupDBFile() string {
	return randFileName("test_", ".db")
}

// call it in defer
func rmFile(name string) {
	os.Remove(name)
}

func MakeBridgeServerConfig(dbfile string) *cmd.BridgeServerConfig {

	// *** prepare objects that aren't string type ***

	// default to regtest
	btcParams := &chaincfg.RegressionNetParams

	// If your Schnorr signer is created separately, load or initialize it here.
	// For this example, we assume you have a local schnorr signer that does that.
	schnorrSigner, err := multisig_client.NewLocalSchnorrSigner([]byte(BTC_CORE_ACCOUNT_PRIV))
	if err != nil {
		fmt.Printf("Error creating schnorr signer: %s", err)
		return nil
	}

	// *** end of preparing objects ***

	return &cmd.BridgeServerConfig{
		// eth side
		EthRpcUrl:          ETH_RPC_URL,
		EthCoreAccountPriv: ETH_BRIDGE_PRIVATE_KEY,
		MSchnorrSigner:     schnorrSigner,
		// state side
		DbFilePath: dbfile,
		// btc side
		BtcRpcServer:       BTC_RPC_SERVER,
		BtcRpcPort:         BTC_RPC_PORT,
		BtcRpcUsername:     BTC_RPC_USERNAME,
		BtcRpcPwd:          BTC_RPC_PWD,
		BtcChainConfig:     btcParams,
		BtcCoreAccountPriv: BTC_CORE_ACCOUNT_PRIV,
		BtcCoreAccountAddr: BTC_CORE_ACCOUNT_ADDR,
		// Http side
		HttpIp:   HTTP_IP,
		HttpPort: HTTP_PORT,
	}
}

func TestEndtoEnd(t *testing.T) {
	logconfig.ConfigDebugLogger()

	sharedcommon.Debug = true
	defer func() {
		sharedcommon.Debug = false
	}()

	// Setup the db file name,
	// this HUGE file is shared as a single db file for btc-side and eth-side state storage.
	db_file_name := setupDBFile()
	defer rmFile(db_file_name)
	t.Logf("db file name: %s", db_file_name)

	// make a bridge server config
	bsc := MakeBridgeServerConfig(db_file_name)
	if bsc == nil {
		t.Fatalf("cannot create bridge server config")
	}

	// Start the bridge server
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	bs, err := cmd.NewBridgeServer(bsc, ctx, &wg)
	if err != nil {
		logger.Fatalf("failed to create bridge server: %v", err)
	}

	t.Logf("bridge server: twbtc contract address: %s", bs.EthEnv.TwbtcContractAddress.Hex())
	t.Logf("bridge server: bridge contract address: %s", bs.EthEnv.BridgeContractAddress.Hex())

	// server is now created and up-running.

	r := bs.BtcRpcClient // TODO: don't use server's btc rpc client, create a separate one for user.

	// *** Setup a http reader to debug ***

	http_reader := reporter.NewHttpReader(HTTP_IP, HTTP_PORT)
	message, err := http_reader.GetHello()
	if err != nil {
		t.Fatalf("cannot get hello from http server %s", err)
	}

	t.Logf("http reader: %s", message)

	if !strings.Contains(message, "world") {
		t.Fatalf("message does not contain 'world'")
	}

	// *** End the setup of http reader ***

	logger.Info("* b2e deposit test *")

	// Send the deposit p2 -> p3
	// Create a sender (p2)
	user_btc_wallet, err := assembler.NewBasicSigner(BTC_USER_ACCOUNT_PRIV, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", BTC_USER_ACCOUNT_PRIV)
	}
	wallet, err := assembler.NewLegacySigner(*user_btc_wallet)
	if err != nil {
		t.Fatalf("cannot create legacy wallet")
	}

	wallet_addr_str := wallet.P2PKH.EncodeAddress()
	logger.WithField("addr", wallet_addr_str).Info("User")

	// Query for UTXOs of (p2)
	// p2 simulates a personal user's wallet.
	utxos, err := r.GetUtxoList(wallet.P2PKH, 1)
	if err != nil {
		t.Fatalf("cannot retrieve utxos with address %s, , error %v", wallet_addr_str, err)
	}
	if len(utxos) == 0 {
		t.Fatalf("no utxos to spend, send some bitcoin to address %s first", wallet_addr_str)
	}
	logger.WithField("count", len(utxos)).Info("User UTXO(s)")

	// List is too long to print
	// for idx, item := range utxos {
	// 	t.Logf("[%d] tx_id: %s, vout: %d, amount: %d, amount_human: %f", idx, item.TxID, item.Vout, item.Amount, item.AmountHuman())
	// }

	// Configure a bridge deposit
	deposit_amount := int64(DEPOSIT_SATOSHI)  // deposit ??? btc
	bridge_address := BTC_CORE_ACCOUNT_ADDR   // bridge wallet
	change_addr := wallet_addr_str            // change is send back to p2 wallet
	fee_amount := int64(WITHDRAW_FEE_SATOSHI) // fee

	// 1) Check balance of bridge wallet
	p3_addr, err := assembler.DecodeAddress(BTC_CORE_ACCOUNT_ADDR, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot decode address %s, error %v", BTC_CORE_ACCOUNT_ADDR, err)
	}

	// log the balance of p3 (before deposit happens)
	p3_balance_1, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", BTC_CORE_ACCOUNT_ADDR, err)
	}

	logger.WithFields(logger.Fields{
		"addr":    BTC_CORE_ACCOUNT_ADDR,
		"satoshi": p3_balance_1,
	}).Info("Bridge")

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), deposit_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	logger.WithField("count", len(selected_utxos)).Info("User UTXOs selected")

	// Craft the [Deposit Tx]
	eth_side_receiver := ETH_USER_ADDR

	logger.WithFields(logger.Fields{
		"amount":   deposit_amount,
		"evm_addr": eth_side_receiver,
		"evm_id":   bs.EthEnv.ChainId.Int64(),
	}).Info("Deposit data")

	tx, err := wallet.MakeBridgeDepositTx(
		selected_utxos,
		bridge_address,
		deposit_amount,
		fee_amount,
		change_addr,
		eth_side_receiver,
		int(bs.EthEnv.ChainId.Int64()),
	)
	if err != nil {
		t.Fatalf("cannot create Tx %v", err)
	}

	logger.WithFields(logger.Fields{
		"TxIn":  len(tx.TxIn),
		"TxOut": len(tx.TxOut),
	}).Info("Craft deposit Tx")

	// Send [Deposit Tx] via RPC
	depositBtcTxHash, err := r.SendRawTx(tx)
	if err != nil {
		t.Fatalf("send raw Tx error, %v", err)
	}

	logger.WithField("txHash", depositBtcTxHash.String()).Info("Tx sent")

	// Generate enough blocks on BTC blockchain to confirm the [Deposit Tx]
	p1_addr, _ := assembler.DecodeAddress(coinbase_legacy_addr_str, assembler.GetRegtestParams())
	r.GenerateBlocks(MAX_BLOCKS, p1_addr)

	// log the balance of p3 (after deposit happens)
	p3_balance_2, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", BTC_CORE_ACCOUNT_ADDR, err)
	}

	logger.WithFields(logger.Fields{
		"addr":    BTC_CORE_ACCOUNT_ADDR,
		"satoshi": p3_balance_2,
	}).Info("Bridge")

	// If balance of p3 is increased, the transfer is successful on the blockchain.
	if p3_balance_2 > p3_balance_1 {
		logger.Info("Deposit mined")
	} else {
		t.Fatalf("Deposit failed")
	}

	// Check on btc monitor side if the deposit is captured
	// 1) Check if the deposit is stored in the deposit storage
	var deposits []btcaction.DepositAction
	for i := 0; i < RETRY_TIMES; i++ {
		deposits, err = bs.MyDepositStorage.GetDepositByTxHash(depositBtcTxHash.String())
		if err != nil {
			t.Fatalf("error retrieving deposit by tx hash: %v", err)
		}
		if len(deposits) > 0 {
			for _, deposit := range deposits {
				logger.WithFields(logger.Fields{
					"tx_hash":  deposit.TxHash,
					"amount":   deposit.DepositValue,
					"receiver": deposit.DepositReceiver,
					"evm_id":   deposit.EvmID,
					"evm_addr": deposit.EvmAddr,
				}).Info("Deposit")
			}
			break
		}
		time.Sleep(3 * time.Second)
	}

	if len(deposits) > 0 {
		logger.Info("Deposit captured")
	} else {
		t.Fatalf("Deposit not captured")
	}

	// Check on http server that deposit is captured.
	resp_deposits, err := http_reader.GetDepositStatus(depositBtcTxHash.String())
	if err != nil {
		t.Fatalf("cannot get deposit status from http server %s", err)
	}

	if len(resp_deposits) > 0 {
		logger.WithFields(logger.Fields{"json": string(resp_deposits)}).Info("http deposit")
	}

	// 2) Check if the UTXO is stored in the UTXO Treasure Vault
	var utxosInVault []btcvault.VaultUTXO
	for i := 0; i < RETRY_TIMES; i++ {
		utxosInVault, err = bs.MyVaultStorage.QueryByTxID(depositBtcTxHash.String())
		if err != nil {
			t.Fatalf("error retrieving utxos from vault storage: %v", err)
		}
		if len(utxosInVault) > 0 {
			for _, utxo := range utxosInVault {
				logger.WithFields(logger.Fields{
					"tx_hash": utxo.TxID,
					"vout":    utxo.Vout,
					"amount":  utxo.Amount,
				}).Info("UTXO")
			}
			break
		}
		time.Sleep(3 * time.Second)
	}

	if len(utxosInVault) > 0 {
		logger.Info("UTXO captured")
	} else {
		t.Fatalf("UTXO not captured")
	}

	// it only takes 0.5 for eth-side mgr to capture the un-minted,
	// and creates a token mint Tx on Ethereum automatcially.
	// so 1 second is long enough.
	time.Sleep(1 * time.Second)

	// Move ethereum blockchain forward to contain the token mint Tx.
	time.Sleep(5 * time.Second)
	// At this step, user's twbtc token balance on eth-side is credited.

	// *** Below begins the EVM -> BTC withdraw process ***

	time.Sleep(1 * time.Second)

	logger.Info("Check if the mint is found on evm")

	time.Sleep(5 * time.Second)

	// Check on http server that deposit status is changed.
	resp_deposits, err = http_reader.GetDepositStatus(depositBtcTxHash.String())
	if err != nil {
		t.Fatalf("cannot get deposit status from http server %s", err)
	}

	if len(resp_deposits) > 0 {
		logger.WithFields(logger.Fields{"json": string(resp_deposits)}).Info("http deposit")
	}

	resp_deposits, err = http_reader.GetDepositStatusByReceiver(eth_side_receiver)
	if err != nil {
		t.Fatalf("cannot get deposit status from http server %s", err)
	}

	if len(resp_deposits) > 0 {
		logger.WithFields(logger.Fields{"json": string(resp_deposits)}).Info("http deposit")
	}

	// logger.Info("* e2b withdraw test *")

	// // Peak the balance of p2, the btc user's balance.
	// p2_addr, _ := assembler.DecodeAddress(btc_user_legacy_addr_str, assembler.GetRegtestParams())
	// p2_balance_before_withdraw, _ := r.GetBalance(p2_addr, 1)

	// logger.WithFields(logger.Fields{
	// 	"balance": p2_balance_before_withdraw,
	// 	"addr":    btc_user_legacy_addr_str,
	// }).Info("User")

	// // Approve Bridge can use twbtc from account[1]
	// ethEnv.sim.Approve(1, REDEEM_SATOSHI)
	// commit()

	// // User request a redeem (p3 bridge, to p2 user)
	// logger.WithFields(logger.Fields{
	// 	"amount": int64(REDEEM_SATOSHI),
	// 	"addr":   btc_user_legacy_addr_str,
	// }).Info("Redeem to")

	// reqTxHash, _ := ethEnv.sim.Request2(ethEnv.sim.GetAuth(1), 1, REDEEM_SATOSHI, btc_user_legacy_addr_str)
	// logger.WithField("txHash", reqTxHash.String()).Info("RedeemRequested")
	// commit()
	// // Give it some time to process the requested redeem
	// time.Sleep(1 * time.Second)

	// // Give it time to let ethtxmanager to prepare the redeem
	// commit()
	// time.Sleep(1 * time.Second)

	// // wait for state to get prepared redeems
	// // loop retry_times, each interval sleep 1 second
	// var redeems []*state.Redeem
	// for i := 0; i < RETRY_TIMES; i++ {
	// 	redeems, err = ethEnv.st.GetPreparedRedeems()
	// 	if err != nil {
	// 		t.Fatalf("error retrieving redeems: %v", err)
	// 	}
	// 	if len(redeems) > 0 {
	// 		break
	// 	}
	// 	time.Sleep(1 * time.Second)
	// }

	// if len(redeems) > 0 {
	// 	for _, redeem := range redeems {
	// 		logger.WithFields(logger.Fields{
	// 			"reqTxHash": redeem.RequestTxHash.String(),
	// 			"preTxHash": redeem.PrepareTxHash.String(),
	// 		}).Info("RedeemPrepared")
	// 	}
	// } else {
	// 	t.Fatalf("RedeemPrepared not captured")
	// }

	// // check on http server that redeem is requested.
	// _requester := eth_side_receiver
	// resp_redeems, err := http_reader.GetRedeemsByRequester(_requester)
	// if err != nil {
	// 	t.Fatalf("cannot get redeems from http server %s", err)
	// }
	// if len(resp_redeems) > 0 {
	// 	logger.WithFields(logger.Fields{"json": string(resp_redeems)}).Info("http redeems")
	// }

	// // Move btc chain forward, to mine the redeem tx
	// r.GenerateBlocks(MAX_BLOCKS, p1_addr)

	// p2_balance_after_withdraw, _ := r.GetBalance(p2_addr, 1)
	// logger.WithFields(logger.Fields{
	// 	"balance": p2_balance_after_withdraw,
	// 	"addr":    btc_user_legacy_addr_str,
	// }).Info("User")

	// // bridge wallet balance (p3) shall decrease
	// p3_balance_3, _ := r.GetBalance(p3_addr, 1)
	// if p3_balance_3 < p3_balance_2 {
	// 	logger.Info("Withdraw mined")
	// } else {
	// 	t.Fatalf("Withdraw failed")
	// }

	// // Allowe some time for the http server to update the status
	// r.GenerateBlocks(1, p1_addr)
	// time.Sleep(3 * time.Second)

	// // check on the http server again, about the btc redeem Tx.
	// resp_redeems, err = http_reader.GetRedeemsByRequester(_requester)
	// if err != nil {
	// 	t.Fatalf("cannot get redeems from http server %s", err)
	// }
	// if len(resp_redeems) > 0 {
	// 	logger.WithFields(logger.Fields{"json": string(resp_redeems)}).Info("http redeems")
	// }

	cancel()  // cancel() signals ctx.Done(), so ends sub go routines politely.
	wg.Wait() // wait for all the routines to be completed then stop the main go routine.
}
