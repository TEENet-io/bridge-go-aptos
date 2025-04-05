package main

import (
	"strconv"
	"testing"
	"time"

	"github.com/TEENet-io/bridge-go/aptosman"
	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/cmd"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/btcsuite/btcd/chaincfg"
	logger "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func convertToPointerSlice(utxos []utxo.UTXO) []*utxo.UTXO {
	utxoPtrs := make([]*utxo.UTXO, len(utxos))
	for i := range utxos {
		utxoPtrs[i] = &utxos[i]
	}
	return utxoPtrs
}

func TestDirectBridgeFlow(t *testing.T) {
	// 初始化 BTC 客户端
	btcUserConfig := &cmd.BtcUserConfig{
		BtcRpcServer:       "127.0.0.1",
		BtcRpcPort:         "19001",
		BtcRpcUsername:     "admin1",
		BtcRpcPwd:          "123",
		BtcChainConfig:     &chaincfg.RegressionNetParams,
		BtcCoreAccountPriv: "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY",
		BtcCoreAccountAddr: "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn",
	}

	// 初始化 Aptos 客户端
	simAptosman, err := aptosman.NewSimAptosman_from_privateKey("0x26f032ddd97e788550f65b8d20f9d037c4330fa27f6f92247f55bd11940774ed")
	require.NoError(t, err)
	aptosClient := simAptosman.Aptosman

	t.Run("Test BTC to Aptos Flow", func(t *testing.T) {
		logger.Infof("开始测试 BTC 到 Aptos 流程\n")
		// 1. 发送 BTC
		amount := int64(100000000) // 0.1 BTC
		bu, _ := cmd.NewBtcUser(btcUserConfig, true)
		balance, _ := bu.GetBalance()
		logger.Infof("当前BTC余额: %.8f BTC\n", float64(balance)/100000000)
		logger.Infof("准备发送 %.8f BTC\n", float64(amount)/100000000)

		txHash, _ := bu.DepositToBridge(amount, 100000, "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih", "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", 1)
		time.Sleep(10 * time.Second)
		rpcclient, _ := rpc.NewRpcClient(&rpc.RpcClientConfig{
			ServerAddr: "127.0.0.1",
			Port:       "19001",
			Username:   "admin1",
			Pwd:        "123",
		})
		addr, _ := assembler.DecodeAddress("mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT", &chaincfg.RegressionNetParams)
		blockHashes, _ := rpcclient.GenerateBlocks(1, addr)
		logger.Infof("生成的区块哈希: %v\n", blockHashes)
		balance, _ = bu.GetBalance()
		time.Sleep(5 * time.Second)
		logger.Infof("BTC 余额: %.8f\n", float64(balance)/100000000)
		// Mint on Aptos
		logger.Infof("开始在 Aptos 上铸币\n")
		TWbalance, _ := aptosClient.GetTWBTCBalance("0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864")
		time.Sleep(8 * time.Second)
		logger.Infof("Aptos 账户余额: %.8f\n", float64(TWbalance)/100000000)
		mintParams := &aptosman.MintParams{
			BtcTxId:  []byte(txHash),
			Amount:   uint64(amount),
			Receiver: "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864",
		}
		_, err = aptosClient.Mint(mintParams)
		client, _ := aptos.NewClient(aptos.DevnetConfig)
		version := ""
		time.Sleep(8 * time.Second)
		mintevent, err := aptosman.GetEvents(client, "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", "mint_events", 10000, 0, "https://api.devnet.aptoslabs.com/")
		// "version":"254385920","guid":{"creation_number":"4","account_address":"0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864"},"sequence_number":"43","type":"0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864::btc_bridgev3::MintEvent","data":
		for _, event := range mintevent {
			version = event["version"].(string)
		}
		logger.Infof("mint_events: %v, %v\n", version, txHash)
		time.Sleep(3 * time.Second)
		require.NoError(t, err)
		logger.Infof("Aptos 铸币成功\n")

		// Verify balance
		TWbalance, _ = aptosClient.GetTWBTCBalance("0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864")
		time.Sleep(3 * time.Second)
		logger.Infof("Aptos 账户余额: %.8f\n", float64(TWbalance)/100000000)
		logger.Infof("BTC 到 Aptos 流程测试完成.\n")
	})

	t.Run("Test Aptos to BTC Flow", func(t *testing.T) {
		logger.Infof("开始测试 Aptos 到 BTC 流程\n")
		time.Sleep(8 * time.Second)
		bu, _ := cmd.NewBtcUser(btcUserConfig, true)
		client, err := aptos.NewClient(aptos.DevnetConfig)
		// 初始化 BTC 客户端
		btcUserConfig_bridge := &cmd.BtcUserConfig{
			BtcRpcServer:       "127.0.0.1",
			BtcRpcPort:         "19001",
			BtcRpcUsername:     "admin1",
			BtcRpcPwd:          "123",
			BtcChainConfig:     &chaincfg.RegressionNetParams,
			BtcCoreAccountPriv: "cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP",
			BtcCoreAccountAddr: "mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih",
		}
		balance, _ := bu.GetBalance()
		bridge_bu, err := cmd.NewBtcUser(btcUserConfig_bridge, true)
		require.NoError(t, err)
		// 在赎回操作前打印余额
		balance, _ = bu.GetBalance()
		logger.Infof("BTC 赎回前余额: %.8f BTC\n", float64(balance)/100000000)
		time.Sleep(1 * time.Second)
		TWbalance, _ := aptosClient.GetTWBTCBalance("0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864")
		logger.Infof("Aptos 赎回前余额: %.8f TWBTC\n", float64(TWbalance)/100000000)

		// Redeem request
		amount := uint64(100000000) // 0.5 BTC
		btcAddr := "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"
		logger.Infof("准备赎回 %.8f BTC 到地址: %s\n", float64(amount)/100000000, btcAddr)

		redeemParams := &aptosman.RequestParams{
			Amount:   amount,
			Receiver: btcAddr,
		}
		time.Sleep(8 * time.Second)
		txHash, err := aptosClient.RedeemRequest(simAptosman.Accounts[0], redeemParams)
		require.NoError(t, err)
		logger.Infof("RedeemRequest请求已发送txHash: %s\n", txHash)

		logger.Infof("等待redeem_request事件确认...\n")
		//GetEvents(client, mouleAddress, "redeem_request_events", limit, start, baseURL)
		time.Sleep(20 * time.Second)
		// 获取赎回请求事件
		events, err := aptosman.GetEvents(client, "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", "redeem_request_events", 10000, 0, "https://api.devnet.aptoslabs.com/")
		require.NoError(t, err)
		logger.Infof("总共获取到 %d 个RedeemRequest请求事件\n", len(events))

		// 解析并比对事件数据
		foundMatchingEvent := false
		latest_amount := ""
		latest_receiver := ""
		latest_sender := ""
		latest_version := ""
		for _, event := range events {
			if version, ok := event["version"]; ok && version != nil {
				latest_version = version.(string)
			}
			if eventData, ok := event["data"].(map[string]interface{}); ok {
				if amount, ok := eventData["amount"]; ok && amount != nil {
					latest_amount = amount.(string)
				}
				if receiver, ok := eventData["receiver"]; ok && receiver != nil {
					latest_receiver = receiver.(string)
				}
				if sender, ok := eventData["sender"]; ok && sender != nil {
					latest_sender = sender.(string)
				}
			}
		}

		if !foundMatchingEvent && latest_amount != "" && latest_receiver != "" && latest_sender != "" {
			amountFloat, _ := strconv.ParseFloat(latest_amount, 64)
			logger.Infof("RedeemRequest amount: %.8f, receiver: %v, sender: %v, version: %v,txHash: %v", amountFloat/100000000, latest_receiver, latest_sender, latest_version, txHash)
		}
		utxos, err := bu.GetUtxos()
		require.NoError(t, err)
		selected_utxos, err := utxo.SelectUtxo(convertToPointerSlice(utxos), int64(amount), 100000)
		if err != nil {
			t.Fatalf("cannot select enough utxos: %v", err)
		}
		prepareParams := &aptosman.PrepareParams{
			RequestTxHash: txHash,
			Requester:     simAptosman.Accounts[0].Address.String(),
			Receiver:      btcAddr,
			Amount:        amount,
			OutpointTxIds: []string{selected_utxos[0].TxID},
			OutpointIdxs:  []uint16{uint16(selected_utxos[0].Vout)},
		}
		time.Sleep(3 * time.Second)
		txHash, err = aptosClient.RedeemPrepare(simAptosman.Accounts[0], prepareParams)
		require.NoError(t, err)
		logger.Infof("RedeemPrepare请求已发送txHash: %s\n", txHash)
		time.Sleep(10 * time.Second)
		//GetEvents(client, mouleAddress, "redeem_prepare_events", limit, start, baseURL)
		redeemEvents, err := aptosman.GetEvents(client, "0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864", "redeem_prepare_events", 10000, 0, "https://api.devnet.aptoslabs.com/")
		require.NoError(t, err)
		logger.Infof("获取到 %d 个redeem_prepare事件\n", len(redeemEvents))
		foundMatchingEvent = false
		for _, event := range redeemEvents {
			if version, ok := event["version"]; ok && version != nil {
				latest_version = version.(string)
			}
			if eventData, ok := event["data"].(map[string]interface{}); ok {
				if amount, ok := eventData["amount"]; ok && amount != nil {
					latest_amount = amount.(string)
				}
				if receiver, ok := eventData["receiver"]; ok && receiver != nil {
					latest_receiver = receiver.(string)
				}
				if sender, ok := eventData["sender"]; ok && sender != nil {
					latest_sender = sender.(string)
				}
			}
		}
		if !foundMatchingEvent && latest_amount != "" && latest_receiver != "" && latest_sender != "" {
			amountFloat, _ := strconv.ParseFloat(latest_amount, 64)
			logger.Infof("RedeemPrepare amount: %.8f, receiver: %v, sender: %v, version: %v,txHash: %v", amountFloat/100000000, latest_receiver, latest_sender, latest_version, txHash)
		}
		// Send BTC
		txHash, _ = bridge_bu.TransferOut(int64(amount), 100000, "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn")
		require.NoError(t, err)
		time.Sleep(5 * time.Second)
		logger.Infof("BTC 提现交易已发送，交易哈希: %s\n", txHash)
		time.Sleep(10 * time.Second)
		rpcclient, _ := rpc.NewRpcClient(&rpc.RpcClientConfig{
			ServerAddr: "127.0.0.1",
			Port:       "19001",
			Username:   "admin1",
			Pwd:        "123",
		})
		addr, _ := assembler.DecodeAddress("mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT", &chaincfg.RegressionNetParams)
		blockHashes, _ := rpcclient.GenerateBlocks(1, addr)
		logger.Infof("生成的区块哈希: %v\n", blockHashes)
		logger.Infof("BTC 提现交易确认成功.\n")
		// 在赎回操作后打印余额
		TWbalance, _ = aptosClient.GetTWBTCBalance("0xfbfe84d58d9ef1366f295066dbf1767f53d52d319843800c63c5e32d66411864")
		logger.Infof("Aptos 赎回后余额: %.8f TWBTC\n", float64(TWbalance)/100000000)
		balance, _ = bu.GetBalance()
		logger.Infof("BTC 赎回后余额: %.8f BTC\n", float64(balance)/100000000)
		logger.Infof("Aptos 到 BTC 流程测试完成.\n")
	})
}
