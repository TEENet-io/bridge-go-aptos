package btcsync

// Notice:
// This test uses a local bitcoin two-node network v0.21.
// And a Ethereum simulation network.

// Test BTC deposit (then mint on eth side)
// 1. Send the deposit
// 2. Monitor captures the deposit
// 3. Monitor Triggers downstream actions.
// 4. Eth manager captures the mint event.
// 5. Eth manager make the mint transaction.
// 6. User gets the minted token.

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"math/big"

	"github.com/TEENet-io/bridge-go/logconfig"
	"github.com/TEENet-io/bridge-go/reporter"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/btctxmanager"
	"github.com/TEENet-io/bridge-go/btcvault"
	sharedcommon "github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/ethtxmanager"
	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/TEENet-io/bridge-go/state"
	"github.com/btcsuite/btcd/chaincfg"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

const (
	RETRY_TIMES         = 10 // retry times for checking the deposit/utxo
	CHANNEL_BUFFER_SIZE = 10

	MAX_BLOCKS = 107 // Generate > 100 blocks to get recognized balance on bitcoin core.
	MIN_BLOCKS = 1   // Minimum step to generate blocks

	// SEND_SATOSHI    = 0.1 * 1e8   // 0.1 btc
	FEE_SATOSHI     = 0.001 * 1e8 // 0.001 btc
	DEPOSIT_SATOSHI = 0.2 * 1e8   // 0.2 btc
	REDEEM_SATOSHI  = 0.1 * 1e8   // 0.1 btc (half of deposit)

	// TEST_EVM_RECEIVER = "0x8ddF05F9A5c488b4973897E278B58895bF87Cb24" // random pick up from etherscan.io
	// TEST_EVM_ID = 1 // eth mainnet

	// This btc wallet holds a lot of money.
	// Also acts the coinbase receiver (block mines and reward goes to this address)
	p1_legacy_priv_key_str = "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"
	p1_legacy_addr_str     = "mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT"

	// user's btc wallet
	p2_legacy_priv_key_str = "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"
	p2_legacy_addr_str     = "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"

	// bridge btc wallet
	p3_legacy_priv_key_str = "cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP"
	p3_legacy_addr_str     = "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih"

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
	EVM_TEST_ACCOUNTS  = 10
	EVM_CHAIN_ID_INT64 = 1337 // Use 1337 as simulated chain id
)

// eth chain
var SimulatedChainID = big.NewInt(EVM_CHAIN_ID_INT64)
var SimulatedEthPrivateKeys = etherman.GenPrivateKeys(EVM_TEST_ACCOUNTS)

// Multisign configuration (remote signer)
// var remoteSignerConfig = mutisig_client.ConnectorConfig{
// 	UserID:        0,
// 	Name:          "client0",
// 	Cert:          "../multisig/config/data/client0.crt",
// 	Key:           "../multisig/config/data/client0.key",
// 	CaCert:        "../multisig/config/data/client0-ca.crt",
// 	ServerAddress: "20.205.130.99:6001",
// 	ServerCACert:  "../multisig/config/data/node0-ca.crt",
// }

// Mutisign (remote signer connector)
// func setupConnector(connConfig mutisig_client.ConnectorConfig) (*mutisig_client.Connector, error) {
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
// 	c, err := mutisig_client.NewConnector(&connConfig)
// 	return c, err
// }

// *** Begin configuration of BTC side ***

// BTC RPC Server configs.
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

// Initial check for btc rpc settings
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

// Setup of BTC RPC client
func setupClient(t *testing.T) (*rpc.RpcClient, error) {
	if !setup() {
		t.Fatal("export env variables first: SERVER, PORT, USER, PASS before running the tests")
	}

	_config := rpc.RpcClientConfig{
		ServerAddr: server,
		Port:       port,
		Username:   username,
		Pwd:        password,
	}
	r, err := rpc.NewRpcClient(&_config)
	if err != nil {
		t.Fatal("cannot create PpcClient with given credentials")
	}
	return r, err
}

// Random file name generator
func randFileName(prefix string, suffix string) string {
	return prefix + ethcommon.Hash(sharedcommon.RandBytes32()).String() + suffix
}

// call it once to get the db file name in this run.
func setupDBFile() string {
	return randFileName("test_", ".db")
}

// call it in defer
func rmFile(name string) {
	os.Remove(name)
}

// Set up the BTC Monitor
func setupBtcMonitor(t *testing.T, r *rpc.RpcClient, st btcaction.RedeemActionStorage, startBlock int) (*BTCMonitor, error) {

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

// *** End configuration of BTC side ***

// *** Begin configuration of ETH side ***

// ETH side facilities
type testEnv struct {
	sim *etherman.SimEtherman

	sqldb   *sql.DB
	statedb *state.StateDB
	st      *state.State
	mgrdb   *ethtxmanager.EthTxManagerDB
	mgr     *ethtxmanager.EthTxManager
	sync    *ethsync.Synchronizer
}

// Setup ETH side facilities
func newTestEnv(t *testing.T, file string, btcChainConfig *chaincfg.Params, btcWallet ethtxmanager.BtcWallet) *testEnv {

	// local schnorr Signer
	ss, err := multisig_client.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("failed to create schnorr wallet: %v", err)
	}

	// or remote schnorr Signer?
	// connector, err := setupConnector(remoteSignerConfig)
	// if err != nil {
	// 	t.Fatalf("failed to create grpc connector: %v", err)
	// }
	// ss := mutisig_client.NewRemoteSchnorrSigner(connector)

	sim, err := etherman.NewSimEtherman(SimulatedEthPrivateKeys, ss, etherman.SimulatedChainID)
	assert.NoError(t, err)

	chainID, err := sim.Etherman.Client().ChainID(context.Background())
	assert.NoError(t, err)

	// create a sql db
	sqldb, err := sql.Open("sqlite3", file)
	assert.NoError(t, err)

	// create a eth2btc state db
	statedb, err := state.NewStateDB(sqldb)
	assert.NoError(t, err)

	// create a eth2btc state from the eth2btc statedb
	st, err := state.New(statedb, &state.StateConfig{ChannelSize: 1, EthChainId: SimulatedChainID})
	assert.NoError(t, err)

	// create a eth tx manager db
	mgrdb, err := ethtxmanager.NewEthTxManagerDB(sqldb)
	assert.NoError(t, err)

	// create a eth synchronizer
	sync, err := ethsync.New(
		sim.Etherman,
		st,
		&ethsync.EthSyncConfig{
			FrequencyToCheckEthFinalizedBlock: frequencyToCheckEthFinalizedBlock,
			BtcChainConfig:                    btcChainConfig,
			EthChainID:                        chainID,
		},
	)
	assert.NoError(t, err)

	// create a eth tx manager
	cfg := &ethtxmanager.EthTxMgrConfig{
		FrequencyToPrepareRedeem:      frequencyToPrepareRedeem,
		FrequencyToMint:               frequencyToMint,
		FrequencyToMonitorPendingTxs:  frequencyToMonitorPendingTxs,
		TimeoutOnWaitingForSignature:  timeoutOnWaitingForSignature,
		TimeoutOnWaitingForOutpoints:  timtoutOnWaitingForOutpoints,
		TimeoutOnMonitoringPendingTxs: timeoutOnMonitoringPendingTxs,
	}
	// TODO change to network-based, multi-party schnorr wallet
	// schnorrWallet := &ethtxmanager.MockSchnorrThresholdWallet{Sim: sim}
	schnorrWallet := ethtxmanager.NewMockedSchnorrAsyncWallet(ss)

	mgr, err := ethtxmanager.NewEthTxManager(cfg, sim.Etherman, statedb, mgrdb, schnorrWallet, btcWallet)
	assert.NoError(t, err)

	return &testEnv{sim, sqldb, statedb, st, mgrdb, mgr, sync}
}

// Close ETH side facilities
func (env *testEnv) close() {
	env.mgrdb.Close()
	env.statedb.Close()
	env.sqldb.Close()
}

// *** End configuration of ETH side ***

func TestDeposit(t *testing.T) {
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

	// 1) Create a UTXO vault storage
	vault_st, err := btcvault.NewVaultSQLiteStorage(db_file_name, p3_legacy_addr_str)
	if err != nil {
		t.Fatalf("cannot create vault storage %v", err)
	}

	// 2) Create a UTXO vault to track a specific btc address
	// This is shared between btc monitor and eth tx manager.
	my_btc_vault := btcvault.NewTreasureVault(p3_legacy_addr_str, vault_st)

	// ** Begins the setup of ETH side ***
	ethEnv := newTestEnv(t, db_file_name, assembler.GetRegtestParams(), my_btc_vault)
	defer ethEnv.close()

	// shortcut to push eth side to mine a block
	commit := ethEnv.sim.Chain.Backend.Commit

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// 1. start eth-side main routines
	wg.Add(1)
	go func() {
		defer wg.Done()
		ethEnv.st.Start(ctx) // state db
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ethEnv.mgr.Start(ctx) // eth-side manager
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		ethEnv.sync.Sync(ctx) // eth-side synchronizer
	}()

	time.Sleep(1 * time.Second)

	// *** Ends the setup of ETH side ***

	// *** Begins the setup of BTC side ***

	// Setup the btc rpc client
	// this rpc instance is shared by btc monitor and btc user (deposit sender)
	r, err := setupClient(t)
	if err != nil {
		t.Fatal("cannot create rpc client")
	}
	defer r.Close()

	// Setup the sqlite db for btc-side tx manager state (for redeem, useless in deposit test)
	btc_mgr_st, err := btcaction.NewSQLiteRedeemStorage(db_file_name)
	if err != nil {
		t.Fatalf("cannot create backend storage %v", err)
	}

	// *** Setup the btc tx manager ***
	// Create a sender (p3)
	bridge_btc_wallet, err := assembler.NewBasicSigner(p3_legacy_priv_key_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", p3_legacy_priv_key_str)
	}
	bridge_wallet, err := assembler.NewLegacySigner(*bridge_btc_wallet)
	if err != nil {
		t.Fatalf("cannot create legacy wallet")
	}

	bridge_wallet_addr_str := bridge_wallet.P2PKH.EncodeAddress()
	// logger.WithField("addr", bridge_wallet_addr_str).Info("Bridge BTC address")

	btcTxMgr := btctxmanager.NewBtcTxManager(my_btc_vault, bridge_wallet, r, ethEnv.st, btc_mgr_st)

	// Turn on evm2btc withdraw loop
	go btcTxMgr.WithdrawLoop()

	// *** End the setup of btc tx manager ***

	// Setup the btc monitor
	latest_height, _ := r.GetLatestBlockHeight()
	// Attention: we start from the latest block height on BTC for clean slate.
	monitor, err := setupBtcMonitor(t, r, btc_mgr_st, int(latest_height))
	if err != nil {
		t.Fatalf("cannot create monitor, %v", err)
	}

	// Setup the observers on btc-side

	// *** Deposit observer ***
	// 1) Create deposit storage
	depo_st, err := btcaction.NewSQLiteDepositStorage(db_file_name)
	if err != nil {
		t.Fatalf("cannot create deposit storage %v", err)
	}
	// 2) Create deposit observer
	d_observer, err := setupObserverDeposit(depo_st)
	if err != nil {
		t.Fatalf("cannot create monitor, %v", err)
	}
	// 3) Deposit Observer start listening to the channel
	go d_observer.GetNotifiedDeposit()

	// 4) Register Deposit Observer to publisher
	monitor.Publisher.RegisterDepositObserver(d_observer.Ch)

	// *** Mint observer ***
	// once a btc deposit occurs, it triggers a mint on eth side.
	// so mint observer is also interested in deposit events.

	// 1) Create mint observer
	mint_observer := NewBTC2EVMObserver(ethEnv.st, CHANNEL_BUFFER_SIZE)

	// 2) Mint Observer start listening to the channel
	go mint_observer.GetNotifiedDeposit()

	// 3) Register Mint Observer to publisher
	monitor.Publisher.RegisterDepositObserver(mint_observer.Ch)

	// *** UTXO Observer ***
	// Setup: UTXO Storage -> UTXO Vault -> UTXO Observer

	// 3) Create a UTXO observer
	// It will stuff the UTXO vault with UTXO(s) once it gets notified
	utxo_observer := NewObserverUTXOVault(my_btc_vault, CHANNEL_BUFFER_SIZE)

	// 4) UTXO Observer starts listening to the channel
	go utxo_observer.GetNotifiedUtxo()

	// 5) Register UTXO Observer to publisher
	monitor.Publisher.RegisterUTXOObserver(utxo_observer.Ch)

	// Turn on the monitor scan loop
	// So it can publish events to observers
	go monitor.ScanLoop()

	// *** Setup a http server to report status ***
	logger.Info("Setup http server to report status")

	http_server := reporter.NewHttpReporter("0.0.0.0", "8080", depo_st, btc_mgr_st, ethEnv.statedb)
	go http_server.Run()
	// Give it some time to start the http server
	time.Sleep(1 * time.Second)

	// *** End the setup of http server ***

	// *** Setup a http reader to debug ***

	http_reader := reporter.NewHttpReader("0.0.0.0", "8080")
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
	user_btc_wallet, err := assembler.NewBasicSigner(p2_legacy_priv_key_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot create wallet from private key %s", p2_legacy_priv_key_str)
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
	deposit_amount := int64(DEPOSIT_SATOSHI) // deposit ??? btc
	bridge_address := p3_legacy_addr_str     // bridge wallet
	change_addr := wallet_addr_str           // change is send back to p2 wallet
	fee_amount := int64(FEE_SATOSHI)         // fee

	// 1) Check balance of bridge wallet
	p3_addr, err := assembler.DecodeAddress(p3_legacy_addr_str, assembler.GetRegtestParams())
	if err != nil {
		t.Fatalf("cannot decode address %s, error %v", p3_legacy_addr_str, err)
	}

	// log the balance of p3 (before deposit happens)
	p3_balance_1, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p3_legacy_addr_str, err)
	}

	logger.WithFields(logger.Fields{
		"addr":    bridge_wallet_addr_str,
		"satoshi": p3_balance_1,
	}).Info("Bridge")

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), deposit_amount, fee_amount)
	if err != nil {
		t.Fatalf("cannot select enough utxos: %v", err)
	}

	logger.WithField("count", len(selected_utxos)).Info("User UTXOs selected")

	// Craft the [Deposit Tx]
	// on EVM side: the receiver is env.Chain.Accounts[1].From
	eth_side_receiver := ethEnv.sim.Chain.Accounts[1].From.String()

	logger.WithFields(logger.Fields{
		"amount":   deposit_amount,
		"evm_addr": eth_side_receiver,
		"evm_id":   EVM_CHAIN_ID_INT64,
	}).Info("Deposit data")

	tx, err := wallet.MakeBridgeDepositTx(
		selected_utxos,
		bridge_address,
		deposit_amount,
		fee_amount,
		change_addr,
		eth_side_receiver,
		EVM_CHAIN_ID_INT64,
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
	p1_addr, _ := assembler.DecodeAddress(p1_legacy_addr_str, assembler.GetRegtestParams())
	r.GenerateBlocks(MAX_BLOCKS, p1_addr)

	// log the balance of p3 (after deposit happens)
	p3_balance_2, err := r.GetBalance(p3_addr, 1)
	if err != nil {
		t.Fatalf("cannot retrieve balance of receiver %s, error %v", p3_legacy_addr_str, err)
	}

	logger.WithFields(logger.Fields{
		"addr":    bridge_wallet_addr_str,
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
		deposits, err = depo_st.GetDepositByTxHash(depositBtcTxHash.String())
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
		utxosInVault, err = vault_st.QueryByTxID(depositBtcTxHash.String())
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
	commit()
	// At this step, user's twbtc token balance on eth-side is credited.

	// *** Below begins the EVM -> BTC withdraw process ***

	time.Sleep(1 * time.Second)

	logger.Info("Check if the mint is found on evm")

	commit()

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

	logger.Info("* e2b withdraw test *")

	// Peak the balance of p2, the btc user's balance.
	p2_addr, _ := assembler.DecodeAddress(p2_legacy_addr_str, assembler.GetRegtestParams())
	p2_balance_before_withdraw, _ := r.GetBalance(p2_addr, 1)

	logger.WithFields(logger.Fields{
		"balance": p2_balance_before_withdraw,
		"addr":    p2_legacy_addr_str,
	}).Info("User")

	// Approve Bridge can use twbtc from account[1]
	ethEnv.sim.Approve(1, REDEEM_SATOSHI)
	commit()

	// User request a redeem (p3 bridge, to p2 user)
	logger.WithFields(logger.Fields{
		"amount": int64(REDEEM_SATOSHI),
		"addr":   p2_legacy_addr_str,
	}).Info("Redeem to")

	reqTxHash, _ := ethEnv.sim.Request2(ethEnv.sim.GetAuth(1), 1, REDEEM_SATOSHI, p2_legacy_addr_str)
	logger.WithField("txHash", reqTxHash.String()).Info("RedeemRequested")
	commit()
	// Give it some time to process the requested redeem
	time.Sleep(1 * time.Second)

	// Give it time to let ethtxmanager to prepare the redeem
	commit()
	time.Sleep(1 * time.Second)

	// wait for state to get prepared redeems
	// loop retry_times, each interval sleep 1 second
	var redeems []*state.Redeem
	for i := 0; i < RETRY_TIMES; i++ {
		redeems, err = ethEnv.st.GetPreparedRedeems()
		if err != nil {
			t.Fatalf("error retrieving redeems: %v", err)
		}
		if len(redeems) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if len(redeems) > 0 {
		for _, redeem := range redeems {
			logger.WithFields(logger.Fields{
				"reqTxHash": redeem.RequestTxHash.String(),
				"preTxHash": redeem.PrepareTxHash.String(),
			}).Info("RedeemPrepared")
		}
	} else {
		t.Fatalf("RedeemPrepared not captured")
	}

	// check on http server that redeem is requested.
	_requester := eth_side_receiver
	resp_redeems, err := http_reader.GetRedeemsByRequester(_requester)
	if err != nil {
		t.Fatalf("cannot get redeems from http server %s", err)
	}
	if len(resp_redeems) > 0 {
		logger.WithFields(logger.Fields{"json": string(resp_redeems)}).Info("http redeems")
	}

	// Move btc chain forward, to mine the redeem tx
	r.GenerateBlocks(MAX_BLOCKS, p1_addr)

	p2_balance_after_withdraw, _ := r.GetBalance(p2_addr, 1)
	logger.WithFields(logger.Fields{
		"balance": p2_balance_after_withdraw,
		"addr":    p2_legacy_addr_str,
	}).Info("User")

	// bridge wallet balance (p3) shall decrease
	p3_balance_3, _ := r.GetBalance(p3_addr, 1)
	if p3_balance_3 < p3_balance_2 {
		logger.Info("Withdraw mined")
	} else {
		t.Fatalf("Withdraw failed")
	}

	// Allowe some time for the http server to update the status
	r.GenerateBlocks(1, p1_addr)
	time.Sleep(3 * time.Second)

	// check on the http server again, about the btc redeem Tx.
	resp_redeems, err = http_reader.GetRedeemsByRequester(_requester)
	if err != nil {
		t.Fatalf("cannot get redeems from http server %s", err)
	}
	if len(resp_redeems) > 0 {
		logger.WithFields(logger.Fields{"json": string(resp_redeems)}).Info("http redeems")
	}

	cancel()  // guess: cancel() ends sub go routines politely.
	wg.Wait() // wait for all the routines to complete.
}
