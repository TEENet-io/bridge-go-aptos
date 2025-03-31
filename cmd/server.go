// Server = eth_side components + btc_side components + db/state + http reporter.
// All components are configured via envionment variables (strings!).

package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	btcrpc "github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcsync"
	"github.com/TEENet-io/bridge-go/btctxmanager"
	"github.com/TEENet-io/bridge-go/btcvault"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/ethtxmanager"
	"github.com/TEENet-io/bridge-go/multisig_client"
	"github.com/TEENet-io/bridge-go/reporter"
	"github.com/TEENet-io/bridge-go/state"
)

// Default params for server.
// More often we don't recommend users to tweak those.
// So we list them here.
const (
	// eth synchronizer config
	frequencyToCheckEthFinalizedBlock = 5 * time.Second

	// eth tx manager config
	frequencyToPrepareRedeem      = 5 * time.Second // read db, gather UTXO, prepare & send RedeemPrepare Tx on ETH side.
	frequencyToMint               = 5 * time.Second // read db, issue Mint Tx on ETH side.
	frequencyToMonitorPendingTxs  = 10 * time.Second
	timeoutOnWaitingForSignature  = 10 * time.Second
	timtoutOnWaitingForOutpoints  = 5 * time.Second // gather UTXOs from BTC wallet.
	timeoutOnMonitoringPendingTxs = 128             // (4x finalized) blocks

	// btc publisher-observer config
	CHANNEL_BUFFER_SIZE = 10
)

// Keep the configuration's fields as "text" as possible.
// Its easier to load it from env vars or a config file.
type BridgeServerConfig struct {
	// eth side
	EthRpcUrl          string                        // json rpc url
	EthCoreAccountPriv string                        // private key of the bridge controlled account
	EthRetroScanBlk    int64                         // retro scan block, tell Sync() to scan from this block, -1 to honor the valude in statedb.
	MSchnorrSigner     multisig_client.SchnorrSigner // remote or local both okay. as long as it can sign() and pub()
	// state side
	DbFilePath string // db file path
	// btc side
	BtcRpcServer       string           // btc rpc server info
	BtcRpcPort         string           // btc rpc server info
	BtcRpcUsername     string           // btc rpc server info
	BtcRpcPwd          string           // btc rpc server info
	BtcChainConfig     *chaincfg.Params // regtest, testnet, mainnet? see btcman/assembler/common.go
	BtcStartBlk        int64            // start block for btc monitor to scan (0=from 0, -1=latest, other=specific block)
	BtcCoreAccountPriv string           // btc core account private key (who sends btc)
	BtcCoreAccountAddr string           // btc core account address (who receives deposit) to be monitored.

	// Http side
	HttpIp   string // eg. 0.0.0.0
	HttpPort string // eg. 8080

	// Predefined ETH smart contract addresses
	PredefinedBridgeContractAddr string // bridge contract address
	PredefinedTwbtcContractAddr  string // twbtc contract address
}

// BridgeServer holds the objects that consists of the bridge server.
type BridgeServer struct {
	// Eth side
	EthEnv *etherman.RealEthChain
	// Eth side: generated objects
	MyEtherman *etherman.Etherman
	MyState    *state.State
	MyStateDb  *state.StateDB
	MyEthMgrDb *ethtxmanager.EthTxManagerDB
	MyEthTxMgr *ethtxmanager.EthTxManager
	MyEthSync  *ethsync.Synchronizer

	// Btc side
	BtcRpcClient *btcrpc.RpcClient
	// Btc side: generated objects
	MyDepositStorage btcaction.DepositStorage
	MyVaultStorage   btcvault.VaultUTXOStorage
	MyBtcVault       *btcvault.TreasureVault
	MyBtcMgr         *btctxmanager.BtcTxManager
	MyBtcMonitor     *btcsync.BTCMonitor
}

// NewBridgeServer creates a new bridge server.
// ctx is used for parental context to cancel the operation of bridge server.
// wg is used to wait for all the goroutines inside the server (monitor, sychronizer, tx manager) to finish.
func NewBridgeServer(bsc *BridgeServerConfig, ctx context.Context, wg *sync.WaitGroup) (*BridgeServer, error) {
	// BTC side config

	// 0) connect to btc network
	myBtcRpcClient, err := SetupBtcRpc(bsc.BtcRpcServer, bsc.BtcRpcPort, bsc.BtcRpcUsername, bsc.BtcRpcPwd)
	if err != nil {
		logger.Fatalf("cannot connect to btc rpc server with %s:%s, %s:%s %v", bsc.BtcRpcServer, bsc.BtcRpcPort, bsc.BtcRpcUsername, bsc.BtcRpcPwd, err)
		return nil, err
	}

	// 1) Create a <UTXO vault storage>
	vaultStorage, err := btcvault.NewVaultSQLiteStorage(bsc.DbFilePath, bsc.BtcCoreAccountAddr)
	if err != nil {
		logger.Fatalf("cannot create vault storage %v", err)
		return nil, err
	}

	// 2) Create a <UTXO vault> over the storage to track a specific btc address
	// This is SHARED between btc monitor and eth tx manager.
	myBtcVault := btcvault.NewTreasureVault(bsc.BtcCoreAccountAddr, vaultStorage)

	// ETH side config

	// 1) Create the real ethereum chain that is terraformed.
	eth_core_account, err := etherman.StringToPrivateKey(bsc.EthCoreAccountPriv)
	if err != nil {
		logger.Fatalf("failed to create core eth account controlled by bridge: %v", err)
		return nil, err
	}
	realEth, err := etherman.NewRealEthChain(bsc.EthRpcUrl, eth_core_account, bsc.MSchnorrSigner, bsc.PredefinedBridgeContractAddr, bsc.PredefinedTwbtcContractAddr)
	if err != nil {
		return nil, err
	}
	logger.WithField("address", realEth.BridgeContractAddress.Hex()).Info("Bridge contract address")
	logger.WithField("address", realEth.TwbtcContractAddress.Hex()).Info("TWBTC contract address")

	// 2) Create the Etherman instance.
	myEtherman, err := etherman.NewEtherman(&etherman.EthermanConfig{
		URL:                   bsc.EthRpcUrl,
		BridgeContractAddress: realEth.BridgeContractAddress,
		TWBTCContractAddress:  realEth.TwbtcContractAddress,
	}, realEth.CoreAccount)

	if err != nil {
		logger.Fatalf("failed to create etherman: %v", err)
		return nil, err
	}

	// Create sql db, and related state_db, state.
	sqldb, err := sql.Open("sqlite3", bsc.DbFilePath)
	if err != nil {
		logger.Fatalf("failed to open db file: %v", err)
		return nil, err
	}

	// state_db
	myStateDb, err := state.NewStateDB(sqldb)
	if err != nil {
		logger.Fatalf("failed to create state db: %v", err)
		return nil, err
	}

	// state
	myState, err := state.New(myStateDb, &state.StateConfig{ChannelSize: 1, UniqueChainId: realEth.ChainId})
	if err != nil {
		logger.Fatalf("failed to create state: %v", err)
		return nil, err
	}

	// eth_tx_manager_db
	myEthTxMgrDb, err := ethtxmanager.NewEthTxManagerDB(sqldb)
	if err != nil {
		logger.Fatalf("failed to create eth tx manager db: %v", err)
		return nil, err
	}

	// eth synchronizer
	// create a eth synchronizer
	myEthSynchronizer, err := ethsync.New(
		myEtherman,
		myState,
		&ethsync.EthSyncConfig{
			FrequencyToCheckEthFinalizedBlock: frequencyToCheckEthFinalizedBlock,
			BtcChainConfig:                    bsc.BtcChainConfig,
			EthChainID:                        realEth.ChainId,
			EthRetroScanBlkNum:                bsc.EthRetroScanBlk,
		},
	)
	if err != nil {
		logger.Fatalf("failed to create eth synchronizer: %v", err)
		return nil, err
	}

	_eth_tx_mgr_cfg := &ethtxmanager.EthTxMgrConfig{
		FrequencyToPrepareRedeem:      frequencyToPrepareRedeem,
		FrequencyToMint:               frequencyToMint,
		FrequencyToMonitorPendingTxs:  frequencyToMonitorPendingTxs,
		TimeoutOnWaitingForSignature:  timeoutOnWaitingForSignature,
		TimeoutOnWaitingForOutpoints:  timtoutOnWaitingForOutpoints,
		TimeoutOnMonitoringPendingTxs: timeoutOnMonitoringPendingTxs,
	}

	// well, eth tx mgr doesn't recognize signer.
	// wrap the signer into "async schnorr wallet".
	_schnorrAsyncWallet := ethtxmanager.NewMockedSchnorrAsyncWallet(bsc.MSchnorrSigner)

	myEthTxMgr, err := ethtxmanager.NewEthTxManager(
		_eth_tx_mgr_cfg,
		myEtherman,
		myStateDb,
		myEthTxMgrDb,
		_schnorrAsyncWallet,
		myBtcVault,
	)
	if err != nil {
		logger.Fatalf("failed to create eth tx manager: %v", err)
		return nil, err
	}

	// Important: Turn on eth-side components!
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := myState.Start(ctx) // state
		if err != nil {
			logger.Fatalf("failed to state eth: %v", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := myEthTxMgr.Start(ctx) // eth-side tx manager
		if err != nil {
			logger.Fatalf("failed to mgr eth: %v", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := myEthSynchronizer.Sync(ctx) // eth-side synchronizer
		if err != nil {
			logger.Fatalf("failed to sync eth: %v", err)
		}
	}()
	// Don't forget to call wg.Wait() in the main routine.

	// Go back to btc side config:
	// 3) Create <Btc Tx Manager Storage> (for redeem purpose, useless in deposit)
	btcMgrStorage, err := btcaction.NewSQLiteRedeemStorage(bsc.DbFilePath)
	if err != nil {
		logger.Fatalf("cannot create backend storage %v", err)
		return nil, err
	}

	// *** Create <btc tx manager> ***
	bridgeBtcCoreAccount, err := assembler.NewNativeSigner(bsc.BtcCoreAccountPriv, bsc.BtcChainConfig)
	if err != nil {
		logger.Fatalf("cannot create wallet from private key %s", bsc.BtcCoreAccountPriv)
		return nil, err
	}
	bridgeBtcOperator, err := assembler.NewNativeOperator(*bridgeBtcCoreAccount)
	if err != nil {
		logger.Fatalf("cannot create legacy wallet")
		return nil, err
	}
	bridgeBtcAddrstr := bridgeBtcOperator.P2PKH.EncodeAddress()
	if bridgeBtcAddrstr != bsc.BtcCoreAccountAddr {
		logger.Fatalf("btc core address mismatch: %s != %s", bridgeBtcAddrstr, bsc.BtcCoreAccountAddr)
	}

	myBtcTxMgr := btctxmanager.NewBtcTxManager(
		myBtcVault,
		bridgeBtcOperator,
		myBtcRpcClient,
		myState,
		btcMgrStorage,
	)

	// Turn on evm2btc withdraw loop
	go myBtcTxMgr.WithdrawLoop()

	// *** Create <btc monitor> for btc2evm deposits ***
	var _start_blk int64
	if bsc.BtcStartBlk == -1 {
		_start_blk, _ = myBtcRpcClient.GetLatestBlockHeight()
	} else {
		_start_blk = bsc.BtcStartBlk
	}

	// Attention: we start from the latest block height on BTC for clean slate.
	myBtcMonitor, err := setupBtcMonitor(myBtcRpcClient, bsc.BtcCoreAccountAddr, btcMgrStorage, int(_start_blk))
	if err != nil {
		logger.Fatalf("cannot create monitor, %v", err)
		return nil, err
	}
	// Can't turn on the monitor loop yet, need to register observers to the monitor loop first.

	// Setup the observers on btc-side

	// *** Deposit observer ***
	// 1) Create <deposit storage>
	depositStorage, err := btcaction.NewSQLiteDepositStorage(bsc.DbFilePath)
	if err != nil {
		logger.Fatalf("cannot create deposit storage %v", err)
		return nil, err
	}
	// 2) Create <deposit observer> over the storage.
	depositObserver, err := setupObserverDeposit(depositStorage)
	if err != nil {
		logger.Fatalf("cannot create deposit observer, %v", err)
		return nil, err
	}
	// 3) Deposit Observer start listening to the channel
	go depositObserver.GetNotifiedDeposit()

	// 4) Register Deposit Observer to publisher
	myBtcMonitor.Publisher.RegisterDepositObserver(depositObserver.Ch)

	// *** Mint observer ***
	// once a btc deposit occurs, it triggers a twbtc mint on eth side.
	// so mint observer is also interested in deposit events.
	// However we don't store mint, it is handled on the eth-side directly.
	// so no storage is allocated for mint observer.

	// 1) Create mint observer
	mintObserver := btcsync.NewBTC2EVMObserver(myState, CHANNEL_BUFFER_SIZE)

	// 2) Mint Observer start listening to the channel
	go mintObserver.GetNotifiedDeposit()

	// 3) Register Mint Observer to publisher
	myBtcMonitor.Publisher.RegisterDepositObserver(mintObserver.Ch)

	// *** UTXO Observer ***
	// Setup: UTXO Storage -> UTXO Vault -> UTXO Observer

	// 3) Create a UTXO observer
	// It will stuff the UTXO vault with UTXO(s) once it gets notified
	utxo_observer := btcsync.NewObserverUTXOVault(myBtcVault, CHANNEL_BUFFER_SIZE)

	// 4) UTXO Observer starts listening to the channel
	go utxo_observer.GetNotifiedUtxo()

	// 5) Register UTXO Observer to publisher
	myBtcMonitor.Publisher.RegisterUTXOObserver(utxo_observer.Ch)

	// Turn on the btc monitor scan loop
	// So it can publish events to observers
	go myBtcMonitor.ScanLoop()

	// *** Setup a http server to report status ***
	// logger.Info("Setup http server to report status")
	http_server := reporter.NewHttpReporter(
		bsc.HttpIp,
		bsc.HttpPort,
		depositStorage,
		btcMgrStorage,
		myStateDb,
	)
	// Turn on the http server
	go http_server.Run()

	// Give it some time to start the http server
	time.Sleep(1 * time.Second)
	// *** End the setup of http server ***

	return &BridgeServer{
		EthEnv:           realEth,
		MyEtherman:       myEtherman,
		MyState:          myState,
		MyStateDb:        myStateDb,
		MyEthMgrDb:       myEthTxMgrDb,
		MyEthTxMgr:       myEthTxMgr,
		MyEthSync:        myEthSynchronizer,
		BtcRpcClient:     myBtcRpcClient,
		MyDepositStorage: depositStorage,
		MyVaultStorage:   vaultStorage,
		MyBtcVault:       myBtcVault,
		MyBtcMgr:         myBtcTxMgr,
		MyBtcMonitor:     myBtcMonitor,
	}, nil
}

// Create, then start the bridge server and wait.
// It contains a prepared bridge server and context + waitgroup.
// Press Ctrl-C to kill the server.
func StartBridgeServerAndWait(bsc *BridgeServerConfig) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // defense programing

	// Set up a signal channel to listen for Ctrl‑C (SIGINT) or SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Launch a new goroutine to handle the signal
	go func() {
		sig := <-sigCh
		fmt.Printf("Received signal: %v, cancelling context...\n", sig)
		cancel()
	}()

	var wg sync.WaitGroup

	_, err := NewBridgeServer(bsc, ctx, &wg)
	if err != nil {
		logger.Fatalf("failed to create bridge server: %v", err)
		return
	}

	// wait for all routines to finish (which is forever)
	wg.Wait()
}

// Helper function. Set up the BTC Monitor, create a new monitor instance
func setupBtcMonitor(r *btcrpc.RpcClient, btcCoreAccountAddr string, st btcaction.RedeemActionStorage, startBlock int) (*btcsync.BTCMonitor, error) {
	monitor, err := btcsync.NewBTCMonitor(
		btcCoreAccountAddr,
		assembler.GetRegtestParams(),
		r,
		int64(startBlock),
		st,
	)
	if err != nil {
		logger.Fatalf("cannot create monitor %v", err)
	}

	return monitor, nil
}

// TODO: remove this helper function.
func setupObserverDeposit(st btcaction.DepositStorage) (*btcsync.ObserverDepositAction, error) {
	return btcsync.NewObserverDepositAction(st, CHANNEL_BUFFER_SIZE), nil
}
