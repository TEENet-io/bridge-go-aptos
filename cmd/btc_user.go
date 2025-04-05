// BtcUser presents an entity that
// 1) Holds user credentials (priv key)
// 2) Perform txns. (deposit to bridge, transfer to other address, etc)
// 2) Monitor user's status. (balance, related deposit, withdraw txns, etc)

package cmd

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcman/assembler"
	btcrpc "github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
)

const (
	BLK_MATURE_OFFSET       = 1 // if a block is BLK_MATURE_OFFSET blocks old, we consider safe.
	REGTEST_COINBASE_ADDR   = "mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT"
	REGTEST_GENERATE_BLOCKS = 101          // Generate 101 blocks in regest.
	REGTEST_FEE_SATOSHI     = 0.0001 * 1e8 // 0.0001 btc = 10,000 satoshi
)

type BtcUserConfig struct {
	BtcRpcServer   string // btc rpc server info
	BtcRpcPort     string // btc rpc server info
	BtcRpcUsername string // btc rpc server info
	BtcRpcPwd      string // btc rpc server info

	BtcChainConfig *chaincfg.Params // regtest, testnet, mainnet? see btcman/assembler/common.go

	BtcCoreAccountPriv string // user's btc private key.
	BtcCoreAccountAddr string // user's btc address.
}

type BtcUser struct {
	BtcRpcClient   *btcrpc.RpcClient         // rpc client
	MyLegacySigner *assembler.NativeOperator // user's signer
	MyAssembler    *assembler.Assembler      // user's btc tx assembler
	MyUserConfig   *BtcUserConfig            // contains a copy of user's config.
}

// Create a new BTC user.
// registerAddress: if true, force register user's address to the bitcoin core rpc node for tracking.
// you can set it to false if using a public btc node because the address is always tracking (I guess).
// For private bitcoin node setup you need to set it to true.
func NewBtcUser(buc *BtcUserConfig, registerAddress bool) (*BtcUser, error) {
	// Set up BTC PRC connection.
	// We use that to interact with BTC blockchain.
	myBtcRpcClient, err := SetupBtcRpc(buc.BtcRpcServer, buc.BtcRpcPort, buc.BtcRpcUsername, buc.BtcRpcPwd)
	if err != nil {
		return nil, err
	}

	// Set up signing entity of user
	// We use that to unlock funds and sign transactions.
	_basic_signer, err := assembler.NewNativeSigner(buc.BtcCoreAccountPriv, buc.BtcChainConfig)
	if err != nil {
		logger.WithField("user_priv", buc.BtcCoreAccountPriv).Error("cannot create Basic Signer by private key")
		return nil, err
	}
	_legacy_signer, err := assembler.NewNativeOperator(*_basic_signer)
	if err != nil {
		logger.Error("cannot create legacy signer from basic signer")
		return nil, err
	}

	_user_addr := _legacy_signer.P2PKH.EncodeAddress()
	// logger.WithField("P2PKH_addr", _user_addr).Info("user btc signer")

	if _user_addr != buc.BtcCoreAccountAddr {
		logger.WithFields(logger.Fields{
			"declared_addr": buc.BtcCoreAccountAddr,
			"actual_addr":   _user_addr,
		}).Error("user btc address mismatch from provided and decoded")
		return nil, err
	}

	if registerAddress {
		// Register user's address to the bitcoin core node.
		// This is necessary for tracking user's balance and transactions.
		// If you are using a public node, you can skip this step.
		err = myBtcRpcClient.ImportAddress(_user_addr, "")
		if err != nil {
			logger.WithField("user_addr", _user_addr).Error("cannot import user address to btc rpc node")
			return nil, err
		}
	}

	return &BtcUser{
		BtcRpcClient:   myBtcRpcClient,
		MyLegacySigner: _legacy_signer,
		MyAssembler:    &assembler.Assembler{ChainConfig: buc.BtcChainConfig, Op: _legacy_signer},
		MyUserConfig:   buc,
	}, nil
}

func (bu *BtcUser) Close() {
	bu.BtcRpcClient.Close() // release rpc connection.
}

func (bu *BtcUser) GetUtxos() ([]utxo.UTXO, error) {
	utxos, err := bu.BtcRpcClient.GetUtxoList(bu.MyLegacySigner.P2PKH, BLK_MATURE_OFFSET)
	if err != nil {
		logger.WithFields(logger.Fields{
			"user_address": bu.MyLegacySigner.P2PKH.EncodeAddress(),
			"error":        err,
		}).Error("cannot retrieve utxos from rpc")
		return nil, err
	}
	if len(utxos) == 0 {
		logger.WithField("user_address", bu.MyLegacySigner.P2PKH.EncodeAddress()).Info("no utxos to spend, send some bitcoin to this address first")
	}
	// logger.WithField("count", len(utxos)).Info("User UTXO(s)")
	return utxos, nil
}

func (bu *BtcUser) GetBalance() (int64, error) {
	balance, err := bu.BtcRpcClient.GetBalance(bu.MyLegacySigner.P2PKH, BLK_MATURE_OFFSET)
	if err != nil {
		logger.WithFields(logger.Fields{
			"user_address": bu.MyLegacySigner.P2PKH.EncodeAddress(),
			"error":        err,
		}).Error("cannot retrieve balance from rpc")
		return 0, err
	}
	// logger.WithField("balance", balance).Info("User Balance")
	return balance, nil
}

// amount: satoshi
// fee: satoshi
func (bu *BtcUser) DepositToBridge(amount int64, feeAmount int64, bridgeAddress string, evmAddr string, evmChainId int) (string, error) {
	// logger.WithFields(logger.Fields{
	// 	"amount":   amount,
	// 	"evm_addr": evmAddr,
	// 	"evm_id":   evmChainId,
	// }).Info("Deposit data")

	// safe guard: check if user has enough balance to make a deposit.
	balance, err := bu.GetBalance()
	if err != nil {
		return "", err
	}
	if balance < amount+feeAmount {
		logger.WithFields(logger.Fields{
			"balance":        balance,
			"requiredAmount": amount + feeAmount,
		}).Error("not enough balance to make a deposit")
		return "", fmt.Errorf("not enough balance: have %d, need %d", balance, amount+feeAmount)
	}

	// Fetch UTXOs
	utxos, err := bu.GetUtxos()
	if err != nil {
		return "", err
	}

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), amount, feeAmount)
	if err != nil {
		logger.WithField("error", err).Error("cannot select enough utxos for deposit")
		return "", err
	}
	// logger.WithField("count", len(selected_utxos)).Info("User UTXOs selected")

	// Craft [Deposit Tx]
	tx, err := bu.MyAssembler.MakeBridgeDepositTx(
		selected_utxos,
		bridgeAddress,
		amount,
		feeAmount,
		bu.MyUserConfig.BtcCoreAccountAddr,
		evmAddr,
		evmChainId,
	)
	if err != nil {
		logger.WithField("error", err).Error("cannot create Tx")
		return "", err
	}

	// logger.WithFields(logger.Fields{
	// 	"TxIn":  len(tx.TxIn),
	// 	"TxOut": len(tx.TxOut),
	// }).Info("Crafted deposit Tx")

	// Send [Deposit Tx] via RPC
	depositBtcTxHash, err := bu.BtcRpcClient.SendRawTx(tx)
	if err != nil {
		logger.WithField("error", err).Error("send raw Tx error")
		return "", err
	}

	// logger.WithField("BTC_TX_ID", depositBtcTxHash.String()).Info("Tx sent")
	return depositBtcTxHash.String(), nil
}

func (bu *BtcUser) TransferOut(amount int64, feeAmount int64, receiverAddr string) (string, error) {
	// logger.WithFields(logger.Fields{
	// 	"amount":       amount,
	// 	"receiverAddr": receiverAddr,
	// }).Info("Transfer btc")

	// safe guard: check if user has enough balance to make a transfer.
	balance, err := bu.GetBalance()
	if err != nil {
		return "", err
	}
	if balance < amount+feeAmount {
		logger.WithFields(logger.Fields{
			"balance":        balance,
			"requiredAmount": amount + feeAmount,
		}).Error("not enough balance to make a transfer")
		return "", fmt.Errorf("not enough balance: have %d, need %d", balance, amount+feeAmount)
	}

	// Fetch UTXOs
	utxos, err := bu.GetUtxos()
	if err != nil {
		return "", err
	}

	// Select barely enough UTXO(s) to spend
	selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), amount, feeAmount)
	if err != nil {
		logger.WithField("error", err).Error("cannot select enough utxos for transfer")
		return "", err
	}
	// logger.WithField("count", len(selected_utxos)).Info("User UTXOs selected")

	// Craft [Transfer Tx]
	tx, err := bu.MyAssembler.MakeTransferOutTx(receiverAddr, amount, bu.MyUserConfig.BtcCoreAccountAddr, feeAmount, selected_utxos)
	if err != nil {
		logger.WithField("error", err).Error("cannot create Tx")
		return "", err
	}

	// logger.WithFields(logger.Fields{
	// 	"TxIn":  len(tx.TxIn),
	// 	"TxOut": len(tx.TxOut),
	// }).Info("Crafted transfer Tx")

	// Send [Transfer Tx] via RPC
	transferBtcTxHash, err := bu.BtcRpcClient.SendRawTx(tx)
	if err != nil {
		logger.WithField("error", err).Error("send raw Tx error")
		return "", err
	}

	// logger.WithField("BTC_TX_ID", transferBtcTxHash.String()).Info("Tx sent")
	return transferBtcTxHash.String(), nil
}

func (bu *BtcUser) MineEnoughBlocks() ([]*chainhash.Hash, error) {
	if bu.MyUserConfig.BtcChainConfig != &chaincfg.RegressionNetParams {
		logger.Error("mine blocks only works in regtest mode")
		return nil, fmt.Errorf("MineEnoughBlocks() only works in btc regtest mode")
	}

	_coinbase_addr, _ := assembler.DecodeAddress(REGTEST_COINBASE_ADDR, bu.MyUserConfig.BtcChainConfig)
	return bu.BtcRpcClient.GenerateBlocks(REGTEST_GENERATE_BLOCKS, _coinbase_addr)
}

// Convert from [] to []*
func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}
