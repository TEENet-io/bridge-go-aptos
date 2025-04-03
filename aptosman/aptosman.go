package aptosman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
	"github.com/btcsuite/btcd/chaincfg"
	logger "github.com/sirupsen/logrus"
)

// Aptosman 是与Aptos区块链交互的核心结构体
type Aptosman struct {
	aptosClient    *aptos.Client
	cfg            *AptosmanConfig
	account        *aptos.Account
	moduleAddress  aptos.AccountAddress
	mu             sync.Mutex
	BtcChainConfig *chaincfg.Params // 添加比特币网络配置

}

// NewAptosman 创建新的Aptosman实例
func NewAptosman(cfg *AptosmanConfig, account *aptos.Account) (*Aptosman, error) {
	// 创建Aptos客户端
	_, networkConfig := GetNetworkConfig(cfg.Network)
	aptosClient, err := aptos.NewClient(networkConfig)
	if err != nil {
		logger.WithField("url", cfg.URL).Errorf("创建Aptos客户端失败: %v", err)
		return nil, err
	}

	// 解析模块地址
	moduleAddress := aptos.AccountAddress{}
	err = moduleAddress.ParseStringRelaxed(cfg.ModuleAddress)
	if err != nil {
		logger.Errorf("解析模块地址失败: %v", err)
		return nil, err
	}

	// 验证模块是否存在
	_, err = aptosClient.AccountResources(moduleAddress)
	if err != nil {
		logger.Errorf("获取模块资源失败: %v", err)
		return nil, err
	}

	return &Aptosman{
		aptosClient:   aptosClient,
		cfg:           cfg,
		account:       account,
		moduleAddress: moduleAddress,
	}, nil
}

// // Client 返回内部Aptos客户端
// func (aptman *Aptosman) Client() *aptos.Client {
// 	return aptman.aptosClient
// }

// GetLatestFinalizedVersion 获取最新确认的区块版本号
func (aptman *Aptosman) GetLatestFinalizedVersion() (uint64, error) {

	// 检查aptman是否为空
	if aptman == nil {
		return 0, fmt.Errorf("aptman实例为空")
	}

	// 检查aptosClient是否为空
	if aptman.aptosClient == nil {
		return 0, fmt.Errorf("aptosClient为空")
	}

	// 获取区块链的最新状态信息
	status, err := aptman.aptosClient.GetProcessorStatus("default_processor")
	if err != nil {
		logger.WithError(err).Error("获取Aptos处理器状态失败")
		return 0, err
	}

	return status, nil
}

// GetModuleEvents 获取一定范围内模块的事件
func (aptman *Aptosman) GetModuleEvents(startVersion, endVersion uint64) (
	[]MintedEvent,
	[]RedeemRequestedEvent,
	[]RedeemPreparedEvent,
	error,
) {
	mintEvents, err := aptman.getMintEvents(startVersion, endVersion)
	if err != nil {
		return nil, nil, nil, err
	}

	redeemRequestedEvents, err := aptman.getRedeemRequestedEvents(startVersion, endVersion)
	if err != nil {
		return nil, nil, nil, err
	}

	redeemPreparedEvents, err := aptman.getRedeemPreparedEvents(startVersion, endVersion)
	if err != nil {
		return nil, nil, nil, err
	}

	return mintEvents, redeemRequestedEvents, redeemPreparedEvents, nil
}

// 获取铸币事件
func (aptman *Aptosman) getMintEvents(startVersion, endVersion uint64) ([]MintedEvent, error) {
	eventHandle := "mint_events"
	events, err := aptman.GetEvents(eventHandle, startVersion, endVersion)
	if err != nil {
		return nil, fmt.Errorf("获取铸币事件失败: %v", err)
	}
	var mintEvents []MintedEvent
	for _, event := range events {
		var mintEvent MintedEvent
		// 直接从原始map解析，实际中应使用标准方法
		if data, ok := event["data"].(map[string]interface{}); ok {
			mintEvent.MintTxHash = event["version"].(string)
			mintEvent.BtcTxId = data["btc_tx_id"].(string)
			mintEvent.Receiver = data["receiver"].(string)
			if amountStr, ok := data["amount"].(string); ok {
				mintEvent.Amount, _ = parseUint64(amountStr)
			}
		}
		mintEvents = append(mintEvents, mintEvent)
	}

	return mintEvents, nil
}

// 获取赎回请求事件
func (aptman *Aptosman) getRedeemRequestedEvents(startVersion, endVersion uint64) ([]RedeemRequestedEvent, error) {
	// eventHandle := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents/redeem_request_events", aptman.moduleAddress.String())
	eventHandle := "redeem_request_events"
	events, err := aptman.GetEvents(eventHandle, startVersion, endVersion)
	if err != nil {
		return nil, fmt.Errorf("获取赎回请求事件失败: %v", err)
	}

	var redeemRequestedEvents []RedeemRequestedEvent
	for _, event := range events {
		var redeemEvent RedeemRequestedEvent
		if data, ok := event["data"].(map[string]interface{}); ok {
			redeemEvent.RequestTxHash = event["version"].(string)
			redeemEvent.Requester = data["sender"].(string)
			redeemEvent.Receiver = data["receiver"].(string)
			if amountStr, ok := data["amount"].(string); ok {
				redeemEvent.Amount, _ = parseUint64(amountStr)
			}
		}
		redeemRequestedEvents = append(redeemRequestedEvents, redeemEvent)
	}

	return redeemRequestedEvents, nil
}

// 获取赎回准备事件
func (aptman *Aptosman) getRedeemPreparedEvents(startVersion, endVersion uint64) ([]RedeemPreparedEvent, error) {
	// eventHandle := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents/redeem_prepare_events", aptman.moduleAddress.String())
	eventHandle := "redeem_prepare_events"
	events, err := aptman.GetEvents(eventHandle, startVersion, endVersion)
	if err != nil {
		return nil, fmt.Errorf("获取赎回准备事件失败: %v", err)
	}

	var redeemPreparedEvents []RedeemPreparedEvent
	for _, event := range events {
		var prepareEvent RedeemPreparedEvent
		if data, ok := event["data"].(map[string]interface{}); ok {
			prepareEvent.RequestTxHash = event["version"].(string)
			prepareEvent.PrepareTxHash = data["eth_tx_hash"].(string)
			prepareEvent.Requester = data["requester"].(string)
			prepareEvent.Receiver = data["receiver"].(string)
			if amountStr, ok := data["amount"].(string); ok {
				prepareEvent.Amount, _ = parseUint64(amountStr)
			}

			// 处理数组类型字段
			if txIds, ok := data["outpoint_tx_ids"].([]interface{}); ok {
				for _, txid := range txIds {
					prepareEvent.OutpointTxIds = append(prepareEvent.OutpointTxIds, txid.(string))
				}
			}

			if idxs, ok := data["outpoint_idxs"].([]interface{}); ok {
				for _, idx := range idxs {
					idxVal, _ := parseUint16(idx.(string))
					prepareEvent.OutpointIdxs = append(prepareEvent.OutpointIdxs, idxVal)
				}
			}
		}
		redeemPreparedEvents = append(redeemPreparedEvents, prepareEvent)
	}

	return redeemPreparedEvents, nil
}

// func (aptman *Aptosman) TWBTCApprove(amount uint64) (string, error) {
// 	aptman.mu.Lock()
// 	defer aptman.mu.Unlock()

// 	// 序列化金额
// 	amountBytes, err := bcs.SerializeU64(amount)
// 	if err != nil {
// 		return "", fmt.Errorf("序列化金额失败: %v", err)
// 	}

// 	// 创建目标地址对象
// 	targetAddress := aptman.moduleAddress

// 	// 正确序列化地址对象
// 	targetAddressBytes, err := bcs.Serialize(&targetAddress)
// 	if err != nil {
// 		return "", fmt.Errorf("序列化目标地址失败: %v", err)
// 	}

// 	// 构建交易Payload
// 	payload := aptos.TransactionPayload{
// 		Payload: &aptos.EntryFunction{
// 			Module: aptos.ModuleId{
// 				Address: aptman.moduleAddress,
// 				Name:    "btc_tokenv3",
// 			},
// 			Function: "transfer",
// 			ArgTypes: []aptos.TypeTag{},
// 			Args: [][]byte{
// 				targetAddressBytes, // 使用正确序列化的地址
// 				amountBytes,
// 			},
// 		},
// 	}

// 	// 构建并提交交易
// 	rawTxn, err := aptman.aptosClient.BuildTransaction(aptman.account.AccountAddress(), payload)
// 	if err != nil {
// 		return "", fmt.Errorf("构建交易失败: %v", err)
// 	}

// 	signedTxn, err := rawTxn.SignedTransaction(aptman.account)
// 	if err != nil {
// 		return "", fmt.Errorf("签名交易失败: %v", err)
// 	}

// 	submitResult, err := aptman.aptosClient.SubmitTransaction(signedTxn)
// 	if err != nil {
// 		return "", fmt.Errorf("提交交易失败: %v", err)
// 	}

// 	txnHash := submitResult.Hash

// 	fmt.Println("txnHash:", txnHash)
// 	// 等待交易完成
// 	_, err = aptman.aptosClient.WaitForTransaction(txnHash)
// 	if err != nil {
// 		return "", fmt.Errorf("等待交易完成失败: %v", err)
// 	}

// 	// 验证交易是否成功
// 	txnInfo, err := aptman.aptosClient.TransactionByHash(txnHash)
// 	if err != nil {
// 		return "", fmt.Errorf("获取交易信息失败: %v", err)
// 	}

// 	userTxn, err := txnInfo.UserTransaction()
// 	if err != nil {
// 		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
// 	}

// 	if userTxn.Success {
// 		return txnHash, nil
// 	} else {
// 		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
// 	}
// }

// func (aptman *Aptosman) mint_tokens(addrStr string, amount uint64) (uint64, error) {
// 	aptman.mu.Lock()
// 	defer aptman.mu.Unlock()

// 	receiverAddr := aptos.AccountAddress{}
// 	err := receiverAddr.ParseStringRelaxed(addrStr)
// 	if err != nil {
// 		return "", fmt.Errorf("解析接收者地址失败: %v", err)
// 	}

// 	amountBytes, err := bcs.SerializeU64(amount)
// 	if err != nil {
// 		return "", fmt.Errorf("序列化金额失败: %v", err)
// 	}

// 	payload := aptos.TransactionPayload{
// 		Payload: &aptos.EntryFunction{
// 			Module: aptos.ModuleId{
// 				Address: aptman.moduleAddress,
// 				Name:    "btc_tokenv3",
// 			},
// 			Function: "mint_tokens",
// 			ArgTypes: []aptos.TypeTag{},
// 			Args: [][]byte{
// 				receiverBytes,
// 				amountBytes,
// 			},
// 		},
// 	}

// 	rawTxn, err := aptman.aptosClient.BuildTransaction(aptman.account.AccountAddress(), payload)

// }

// 铸造TWBTC代币
func (aptman *Aptosman) Mint(params *MintParams) (string, error) {
	aptman.mu.Lock()
	defer aptman.mu.Unlock()

	// 序列化参数
	receiverAddr := aptos.AccountAddress{}
	err := receiverAddr.ParseStringRelaxed(params.Receiver)
	if err != nil {
		return "", fmt.Errorf("解析接收者地址失败: %v", err)
	}

	btc_tx_id_len := len(params.BtcTxId)
	btc_tx_id_bytes := make([]byte, 0)

	if btc_tx_id_len < 128 {
		btc_tx_id_bytes = append(btc_tx_id_bytes, byte(btc_tx_id_len))
	} else {
		encodedLen := byte(btc_tx_id_len) | 0x80
		btc_tx_id_bytes = append(btc_tx_id_bytes, encodedLen, byte(btc_tx_id_len>>7))
	}

	btc_tx_id_bytes = append(btc_tx_id_bytes, []byte(params.BtcTxId)...)

	receiverBytes, err := bcs.Serialize(&receiverAddr)
	if err != nil {
		return "", fmt.Errorf("序列化接收者地址失败: %v", err)
	}

	amountBytes, err := bcs.SerializeU64(params.Amount)
	if err != nil {
		return "", fmt.Errorf("序列化金额失败: %v", err)
	}

	// 构建交易Payload
	payload := aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: aptman.moduleAddress,
				Name:    "btc_bridgev3",
			},
			Function: "mint",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{
				btc_tx_id_bytes,
				receiverBytes,
				amountBytes,
			},
		},
	}

	// 构建、签名并提交交易
	txn, err := aptman.aptosClient.BuildTransaction(aptman.account.AccountAddress(), payload)
	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}

	signedTxn, err := txn.SignedTransaction(aptman.account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := aptman.aptosClient.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}

	// 等待交易确认
	_, err = aptman.aptosClient.WaitForTransaction(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
	}
	// TODO check resullt
	return submitResult.Hash, nil
}

// RedeemRequest 发起赎回请求
func (aptman *Aptosman) RedeemRequest(account *aptos.Account, params *RequestParams) (string, error) {
	// moduleAddr := aptman.moduleAddress.String()

	// 序列化参数
	amountBytes, err := bcs.SerializeU64(params.Amount)
	if err != nil {
		return "", fmt.Errorf("序列化金额失败: %v", err)
	}

	receiverBytes := serializeString(params.Receiver)

	// 构建交易Payload
	payload := aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: aptman.moduleAddress,
				Name:    "btc_bridgev3",
			},
			Function: "redeem_request",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{
				amountBytes,
				receiverBytes,
			},
		},
	}

	// 构建、签名并提交交易
	txn, err := aptman.aptosClient.BuildTransaction(account.AccountAddress(), payload)
	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}

	signedTxn, err := txn.SignedTransaction(account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := aptman.aptosClient.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}

	// 等待交易确认
	_, err = aptman.aptosClient.WaitForTransaction(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
	}

	return submitResult.Hash, nil
}

func (aptman *Aptosman) RegisterTWBTC(account *aptos.Account) (string, error) {
	// 构建函数调用

	// 使用更简单的方式构建交易
	rawTxn, err := aptman.aptosClient.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: aptman.moduleAddress,
				Name:    "btc_tokenv3",
			},
			Function: "register",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{},
		},
	})

	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}

	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := aptman.aptosClient.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}

	// 等待交易确认
	_, err = aptman.aptosClient.WaitForTransaction(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
	}

	// 验证交易是否成功
	txnInfo, err := aptman.aptosClient.TransactionByHash(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}

	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}

	if !userTxn.Success {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}

	return submitResult.Hash, nil
}

// RedeemPrepare 准备赎回交易
func (aptman *Aptosman) RedeemPrepare(account *aptos.Account, params *PrepareParams) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(aptman.moduleAddress.String())
	if err != nil {
		return "", fmt.Errorf("解析模块地址失败: %v", err)
	}

	txHashBytes := serializeString(params.RequestTxHash)

	requesterAddr := aptos.AccountAddress{}
	err = requesterAddr.ParseStringRelaxed(params.Requester)
	if err != nil {
		return "", fmt.Errorf("解析requester地址失败: %v", err)
	}
	requesterBytes, err := bcs.Serialize(&requesterAddr)
	if err != nil {
		return "", fmt.Errorf("序列化requester地址失败: %v", err)
	}

	receiverBytes := serializeString(params.Receiver)

	amountBytes, err := bcs.SerializeU64(params.Amount)
	if err != nil {
		return "", fmt.Errorf("序列化金额失败: %v", err)
	}

	outpointTxIdsBytes := serializeStringVector(params.OutpointTxIds)

	uint64Idxs := make([]uint64, len(params.OutpointIdxs))
	for i, idx := range params.OutpointIdxs {
		uint64Idxs[i] = uint64(idx)
	}
	outpointIdxsBytes := serializeU64Vector(uint64Idxs)

	// 构建交易
	rawTxn, err := aptman.aptosClient.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: aptman.moduleAddress,
				Name:    "btc_bridgev3",
			},
			Function: "redeem_prepare",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{txHashBytes, requesterBytes, receiverBytes, amountBytes, outpointTxIdsBytes, outpointIdxsBytes},
		},
	})

	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}

	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := aptman.aptosClient.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}
	fmt.Println("submitResult.Hash:", submitResult.Hash)
	// 等待交易确认
	_, err = aptman.aptosClient.WaitForTransaction(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
	}

	// 验证交易是否成功
	txnInfo, err := aptman.aptosClient.TransactionByHash(submitResult.Hash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}

	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}

	if !userTxn.Success {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}

	return submitResult.Hash, nil
}

// 获取TWBTC余额
func (aptman *Aptosman) GetTWBTCBalance(addrStr string) (uint64, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(addrStr)
	if err != nil {
		return 0, fmt.Errorf("解析地址失败: %v", err)
	}

	resources, err := aptman.aptosClient.AccountResources(address)
	if err != nil {
		return 0, fmt.Errorf("获取账户资源失败: %v", err)
	}

	// 构建资源类型
	resourceType := fmt.Sprintf("0x1::coin::CoinStore<%s::btc_tokenv3::BTC>", aptman.moduleAddress.String())

	for _, resource := range resources {
		if resource.Type == resourceType {
			if coinData, ok := resource.Data["coin"]; ok {
				if coinMap, ok := coinData.(map[string]interface{}); ok {
					if valueStr, ok := coinMap["value"].(string); ok {
						return parseUint64(valueStr)
					}
				}
			}
			break
		}
	}

	return 0, nil
}

// IsPrepared checks if a redemption is prepared
func (aptman *Aptosman) IsPrepared(txHash string) (bool, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(aptman.moduleAddress.String())
	if err != nil {
		return false, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取桥配置资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::PreparedRedeems", address.String())
	resource, err := aptman.aptosClient.AccountResource(address, resourceType)
	if err != nil {
		return false, fmt.Errorf("获取PreparedRedeems资源失败: %v", err)
	}

	// 检查交易哈希是否在已准备列表中
	if data, ok := resource["data"].(map[string]interface{}); ok {
		fmt.Println("data", data)
		if prepared, ok := data["prepared"].([]interface{}); ok {
			for _, hash := range prepared {
				if hash.(string) == txHash {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// IsMinted 检查比特币交易是否已铸币
func (aptman *Aptosman) IsMinted(btcTxId string) (bool, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(aptman.moduleAddress.String())
	if err != nil {
		return false, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取已使用的比特币交易ID资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::UsedBtcTxIds", address.String())
	resource, err := aptman.aptosClient.AccountResource(address, resourceType)
	if err != nil {
		return false, fmt.Errorf("获取UsedBtcTxIds资源失败: %v", err)
	}

	// 检查交易ID是否在used列表中
	if data, ok := resource["data"].(map[string]interface{}); ok {
		if used, ok := data["used"].([]interface{}); ok {
			for _, id := range used {
				if id.(string) == btcTxId {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// IsUsed 检查比特币交易是否已使用
func (aptman *Aptosman) IsUsed(btcTxId string) (bool, error) {
	// 在Aptos中，IsMinted和IsUsed功能相同，都是检查交易ID是否已使用
	return aptman.IsMinted(btcTxId)
}

// https://api.devnet.aptoslabs.com/v1/accounts/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d/events/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents/redeem_request_events?start=1

// https://fullnode.devnet.aptoslabs.com/v1/accounts/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d/events/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents/mint_events
// GetEvents 获取事件
func (aptman *Aptosman) GetEvents(events_field string, limit uint64, start uint64) ([]map[string]interface{}, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(aptman.moduleAddress.String())
	if err != nil {
		return nil, fmt.Errorf("Parse module address failed: %v", err)
	}

	// 获取BridgeEvents资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	resource, err := aptman.aptosClient.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("Get bridge events resource failed: %v", err)
	}
	// 从资源数据中获取redeem_request_events
	data, ok := resource["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Resource data format is incorrect")
	}

	eventsData, ok := data[events_field]
	if !ok {
		dataBytes, _ := json.MarshalIndent(data, "", "  ")
		return nil, fmt.Errorf("Field '%s' not found in contract data: %s", events_field, string(dataBytes))
	}

	// 解析事件数据
	eventsMap, ok := eventsData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Resource data format is incorrect")
	}

	// 获取事件计数器
	counterStr, ok := eventsMap["counter"].(string)
	if !ok {
		return nil, fmt.Errorf("未找到counter字段")
	}

	counter, ok := new(big.Int).SetString(counterStr, 10)
	if !ok {
		return nil, fmt.Errorf("counter字段解析失败")
	}

	// 如果没有事件，直接返回空数组
	if counter.Cmp(big.NewInt(0)) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 获取事件句柄信息
	guidData, ok := eventsMap["guid"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("未找到guid字段")
	}

	idData, ok := guidData["id"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("未找到id字段")
	}

	// 获取账户地址
	accountAddressStr, ok := idData["addr"].(string)
	if !ok {
		return nil, fmt.Errorf("未找到addr字段")
	}

	// 解析账户地址
	accountAddress := aptos.AccountAddress{}
	err = accountAddress.ParseStringRelaxed(accountAddressStr)
	if err != nil {
		return nil, fmt.Errorf("解析账户地址失败: %v", err)
	}
	// 使用硬编码的API URL
	network := aptman.cfg.Network
	baseURL, _ := GetNetworkConfig(network)

	// 正确构建事件API路径
	fullURL := fmt.Sprintf("%s/accounts/%s/events/%s/%s",
		baseURL,
		accountAddress.String(),
		resourceType,
		events_field)

	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	q := req.URL.Query()
	// if limit > 0 {
	// 	q.Add("limit", fmt.Sprintf("%d", limit))
	// }
	// if start > 0 {
	// 	q.Add("start", fmt.Sprintf("%d", start))
	// }
	req.URL.RawQuery = q.Encode()

	// 发送请求
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败: 状态码 %d, 响应体: %s", resp.StatusCode, string(body))
	}
	// 解析响应
	var events []map[string]interface{}

	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return events, nil
}

// 辅助函数：解析字符串为uint64
func parseUint64(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

// 辅助函数：解析字符串为uint16
func parseUint16(s string) (uint16, error) {
	val, err := strconv.ParseUint(s, 10, 16)
	return uint16(val), err
}

// 辅助函数：创建字符串的BCS表示
func bcsEncodeString(s string) []byte {
	// 字符串长度 + 内容
	result := make([]byte, 0, len(s)+1)
	result = append(result, byte(len(s)))
	result = append(result, []byte(s)...)
	return result
}

func (aptman *Aptosman) MintTokensToContract(amount uint64) (string, error) {
	// 准备铸币参数
	btcTxId := []byte(fmt.Sprintf("admin_mint_%d", time.Now().Unix()))
	params := &MintParams{
		BtcTxId:  btcTxId,
		Amount:   amount,
		Receiver: aptman.moduleAddress.String(), // 将代币直接铸造给合约地址
	}

	// 执行铸币操作
	return aptman.Mint(params)
}

// NewSyncWorker 创建一个新的同步工作器
func (aptman *Aptosman) NewSyncWorker() *AptosSyncWorker {
	return NewAptosSyncWorker(aptman)
}

// NewMgrWorker 创建一个新的管理工作器
func (aptman *Aptosman) NewMgrWorker() *AptosMgrWorker {
	return NewAptosMgrWorker(aptman)
}
