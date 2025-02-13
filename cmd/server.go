// Server = eth_side components + btc_side components + db/state + http reporter.
// All components are configured via envionment variables.

package cmd

import (
	"database/sql"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcvault"
	"github.com/TEENet-io/bridge-go/etherman"
	"github.com/TEENet-io/bridge-go/ethsync"
	"github.com/TEENet-io/bridge-go/ethtxmanager"
	"github.com/TEENet-io/bridge-go/multisig"
	"github.com/TEENet-io/bridge-go/state"
)

// Default params for server.
const (
	// eth synchronizer
	frequencyToCheckEthFinalizedBlock = 100 * time.Millisecond

	// eth tx manager config
	frequencyToPrepareRedeem      = 500 * time.Millisecond
	frequencyToMint               = 500 * time.Millisecond // 0.5 second
	frequencyToMonitorPendingTxs  = 500 * time.Millisecond
	timeoutOnWaitingForSignature  = 1 * time.Second
	timtoutOnWaitingForOutpoints  = 1 * time.Second
	timeoutOnMonitoringPendingTxs = 10 // blocks
)

// Keep the configuration's fields as "text" as possible.
type BridgeServerConfig struct {
	// eth side
	EthRpcUrl          string                 // json rpc url
	EthCoreAccountPriv string                 // private key of the bridge controlled account
	schnorrSigner      multisig.SchnorrSigner // remote or local both okay. as long as it can sign() and pub()
	// state side
	DbFilePath string // db file path
	// btc side
	BtcCoreAccountAddr string           // btc core account address (who receives deposit) to be monitored.
	BtcChainConfig     *chaincfg.Params // regtest, testnet, mainnet? see btcman/assembler/common.go
}

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
}

func NewBridgeServer(bsc *BridgeServerConfig) (*BridgeServer, error) {
	// BTC side config
	// 1) Create a <UTXO vault storage>
	vault_st, err := btcvault.NewSQLiteStorage(bsc.DbFilePath, bsc.BtcCoreAccountAddr)
	if err != nil {
		logger.Fatalf("cannot create vault storage %v", err)
		return nil, err
	}

	// 2) Create a <UTXO vault> over the storage to track a specific btc address
	// This is shared between btc monitor AND eth tx manager.
	my_btc_vault := btcvault.NewTreasureVault(bsc.BtcCoreAccountAddr, vault_st)

	// 1) Create the real ethereum chain that is terraformed.
	eth_core_account, err := etherman.StringToPrivateKey(bsc.EthCoreAccountPriv)
	if err != nil {
		logger.Fatalf("failed to create core eth account controlled by bridge: %v", err)
		return nil, err
	}
	realEth, err := etherman.NewRealEthChain(bsc.EthRpcUrl, eth_core_account, bsc.schnorrSigner)
	if err != nil {
		return nil, err
	}

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
	myState, err := state.New(myStateDb, &state.StateConfig{ChannelSize: 1, EthChainId: realEth.ChainId})
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
	_schnorrAsyncWallet := ethtxmanager.NewMockedSchnorrAsyncWallet(bsc.schnorrSigner)

	myEthTxMgr, err := ethtxmanager.NewEthTxManager(
		_eth_tx_mgr_cfg,
		myEtherman,
		myStateDb,
		myEthTxMgrDb,
		_schnorrAsyncWallet,
		my_btc_vault,
	)
	if err != nil {
		logger.Fatalf("failed to create eth tx manager: %v", err)
		return nil, err
	}

	return &BridgeServer{
		EthEnv:     realEth,
		MyEtherman: myEtherman,
		MyState:    myState,
		MyStateDb:  myStateDb,
		MyEthMgrDb: myEthTxMgrDb,
		MyEthTxMgr: myEthTxMgr,
		MyEthSync:  myEthSynchronizer,
	}, nil
}
