package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"io/ioutil"
	"time"
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/aptos-labs/aptos-go-sdk/bcs"
)

// CheckTWBTCBalance checks the TWBTC token balance for an account
func CheckTWBTCBalance(client *aptos.Client, address aptos.AccountAddress, moduleAddress string) (*big.Int, error) {
	resources, err := client.AccountResources(address)
	if err != nil {
		return nil, fmt.Errorf("获取账户资源失败: %v", err)
	}

	moduleAddr := aptos.AccountAddress{}
	err = moduleAddr.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	resourceType := fmt.Sprintf("0x1::coin::CoinStore<%s::btc_tokenv3::BTC>", moduleAddr.String())
	
	var balance *big.Int = big.NewInt(0)
	for _, resource := range resources {
		if resource.Type == resourceType {
			if coinData, ok := resource.Data["coin"]; ok {
				if coinMap, ok := coinData.(map[string]interface{}); ok {
					if valueStr, ok := coinMap["value"].(string); ok {
						balance, _ = new(big.Int).SetString(valueStr, 10)
					}
				}
			}
			break
		}
	}

	return balance, nil
}

// // SendTWBTC sends TWBTC tokens to another account
// func SendTWBTC(client *aptos.Client, senderAccount aptos.TransactionSigner, receiverAddress aptos.AccountAddress, amount uint64, moduleAddress string) (string, error) {
// 	// toodo
// 	return "", nil
// }



func initTWBTC(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}

	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_tokenv3",
			},
			Function: "initialize_module",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{},
		},
	},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}

	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}
	// 检查交易状态
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}
	if userTxn.Success {
		return txnHash, nil
	} else {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}
	return txnHash, nil
}


func initBridge(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, feeAccount aptos.AccountAddress, fee uint64) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}
	// admin: &signer,
	// // pk: vector<u8>,
	// fee_account: address,
	// fee: u64
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}
	
	// Convert feeAccount.Address to []byte
	feeAccountBytes, err := bcs.Serialize(&feeAccount)
	if err != nil {
		return "", fmt.Errorf("序列化费用账户地址失败: %v", err)
	}
	
	// Convert fee to []byte
	feeBytes, err := bcs.SerializeU64(fee)
	if err != nil {
		return "", fmt.Errorf("序列化费用失败: %v", err)
	}
	
	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_bridgev3",
			},
			Function: "initialize",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{feeAccountBytes, feeBytes},
		},
	},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}

	// 验证交易是否成功	
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)	
	}
	if userTxn.Success {
		return txnHash, nil
	} else {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}


	return txnHash, nil

}

func registerTWBTC(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress aptos.AccountAddress) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}

	receiverAddressBytes, err := bcs.Serialize(&receiverAddress)
	if err != nil {
		return "", fmt.Errorf("序列化接收方地址失败: %v", err)
	}

	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_tokenv3",
			},
			Function: "registerv2",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{receiverAddressBytes},
		},
	},
	)

	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}

	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}
	if userTxn.Success {
		return txnHash, nil
	} else {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}

	return txnHash, nil


}


func mintTWBTC(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress aptos.AccountAddress, amount uint64, btc_tx_id string) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}

	// btc_tx_id: String,
	// receiver: address,
	// amount: u64,


	// 正确的BCS编码方式 - 首先需要编码字符串长度，然后是字符串内容
	btc_tx_id_len := len(btc_tx_id)
	btc_tx_id_bytes := make([]byte, 0)

	// 添加字符串长度(使用LEB128编码)
	if btc_tx_id_len < 128 {
		btc_tx_id_bytes = append(btc_tx_id_bytes, byte(btc_tx_id_len))
	} else {
		// 处理更长的字符串...
		// 这里简化处理，实际上应该使用LEB128编码
		encodedLen := byte(btc_tx_id_len) | 0x80
		btc_tx_id_bytes = append(btc_tx_id_bytes, encodedLen, byte(btc_tx_id_len>>7))
	}

	// 添加字符串内容
	btc_tx_id_bytes = append(btc_tx_id_bytes, []byte(btc_tx_id)...)


	receiverAddressBytes, err := bcs.Serialize(&receiverAddress)
	if err != nil {	
		return "", fmt.Errorf("序列化接收方地址失败: %v", err)
	}
	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return "", fmt.Errorf("序列化金额失败: %v", err)
	}

	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_bridgev3",
			},
			Function: "mint",
			ArgTypes: []aptos.TypeTag{},
			Args: [][]byte{btc_tx_id_bytes, receiverAddressBytes, amountBytes},
		},
	},
	)
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}

	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}
	if userTxn.Success {
		return txnHash, nil
	} else {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}

	return txnHash, nil
}


func redeemRequest(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress string, amount uint64) (string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return "", fmt.Errorf("解析地址失败: %v", err)
	}
	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return "", fmt.Errorf("序列化金额失败: %v", err)
	}
	

	// 正确的BCS编码方式 - 首先需要编码字符串长度，然后是字符串内容
	receiverStrLen := len(receiverAddress)
	receiverBytes := make([]byte, 0)

	// 添加字符串长度(使用LEB128编码)
	if receiverStrLen < 128 {
		receiverBytes = append(receiverBytes, byte(receiverStrLen))
	} else {
		// 处理更长的字符串...
		// 这里简化处理，实际上应该使用LEB128编码
		encodedLen := byte(receiverStrLen) | 0x80
		receiverBytes = append(receiverBytes, encodedLen, byte(receiverStrLen>>7))
	}

	// 添加字符串内容
	receiverBytes = append(receiverBytes, []byte(receiverAddress)...)

	// 使用正确编码的String类型调用合约
	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_bridgev3",
			},
			Function: "redeem_request",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{amountBytes, receiverBytes},
		},
	},
	)

	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return "", fmt.Errorf("获取交易信息失败: %v", err)
	}
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return "", fmt.Errorf("解析用户交易信息失败: %v", err)
	}
	if userTxn.Success {
		return txnHash, nil
	} else {
		return "", fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}



	return txnHash, nil
}

// 定义事件结构体，与Move合约中的事件结构匹配
// TokenMintEvent 对应 btc_tokenv3::MintEvent
type TokenMintEvent struct {
	Amount    uint64 `json:"amount"`
	Recipient string `json:"recipient"`
	BtcTxId   string `json:"btc_txid"`
}

// TokenBurnEvent 对应 btc_tokenv3::BurnEvent
type TokenBurnEvent struct {
	Amount     uint64 `json:"amount"`
	Burner     string `json:"burner"`
	BtcAddress string `json:"btc_address"`
}

// BridgeMintEvent 对应 btc_bridgev3::MintEvent
type BridgeMintEvent struct {
	BtcTxId  string `json:"btc_tx_id"`
	Receiver string `json:"receiver"`
	Amount   uint64 `json:"amount"`
}

// RedeemRequestEvent 对应 btc_bridgev3::RedeemRequestEvent
type RedeemRequestEvent struct {
	Sender   string `json:"sender"`
	Amount   uint64 `json:"amount"`
	Receiver string `json:"receiver"`
}

// RedeemPrepareEvent 对应 btc_bridgev3::RedeemPrepareEvent
type RedeemPrepareEvent struct {
	EthTxHash      string   `json:"eth_tx_hash"`
	Requester      string   `json:"requester"`
	Receiver       string   `json:"receiver"`
	Amount         uint64   `json:"amount"`
	OutpointTxIds  []string `json:"outpoint_tx_ids"`
	OutpointIdxs   []uint64 `json:"outpoint_idxs"`
}

// GetBridgeConfig 获取桥的配置信息
func GetBridgeConfig(client *aptos.Client, moduleAddress string) (map[string]interface{}, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取BridgeConfig资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeConfig", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取桥配置失败: %v", err)
	}

	return resource, nil
}

// GetBridgeMintEvents 获取桥铸币事件
func GetBridgeMintEvents(client *aptos.Client, moduleAddress string, limit uint64) ([]BridgeMintEvent, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取BridgeEvents资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取桥事件资源失败: %v", err)
	}
	// fmt.Println(resource) 
	// map[data:map[mint_events:map[counter:1 guid:map[id:map[addr:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d creation_num:8]]] redeem_prepare_events:map[counter:0 guid:map[id:map[addr:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d creation_num:10]]] redeem_request_events:map[counter:2 guid:map[id:map[addr:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d creation_num:9]]]] type:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents]
	// 直接访问map
	eventsData, ok := resource["data"].(map[string]interface{})["mint_events"]
	if !ok {
		return nil, fmt.Errorf("资源中未找到mint_events字段")
	}

	eventsMap, ok := eventsData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mint_events字段格式不正确")
	}

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
		return []BridgeMintEvent{}, nil
	}

	// 计算查询范围
	start := uint64(0)
	if counter.Uint64() > limit {
		start = counter.Uint64() - limit
	}

	// 获取事件句柄
	guidStr, ok := eventsMap["guid"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("未找到guid字段")
	}

	// creationNumStr, ok := guidStr["id"].(map[string]interface{})["creation_num"].(string)
	// if !ok {
	// 	return nil, fmt.Errorf("未找到creation_num字段")
	// }

	accountAddressStr, ok := guidStr["id"].(map[string]interface{})["addr"].(string)
	if !ok {
		return nil, fmt.Errorf("未找到addr字段")
	}

	accountAddress := aptos.AccountAddress{}
	err = accountAddress.ParseStringRelaxed(accountAddressStr)
	if err != nil {
		return nil, fmt.Errorf("解析账户地址失败: %v", err)
	}

	// creationNum, ok := new(big.Int).SetString(creationNumStr, 10)
	// if !ok {
	// 	return nil, fmt.Errorf("creation_num字段解析失败")
	// }

	// 使用硬编码的API URL
	baseURL := "https://fullnode.devnet.aptoslabs.com" // 默认使用devnet
	
	// 构建完整的URL - 使用Aptos REST API格式
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s", 
		baseURL,
		accountAddress.String(),
		resourceType + "/mint_events")
	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	q := req.URL.Query()
	q.Add("start", fmt.Sprintf("%d", start))
	q.Add("limit", fmt.Sprintf("%d", limit))
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
	var events []struct {
		Data    map[string]interface{} `json:"data"`
		Version string                 `json:"version"`
		Guid    map[string]interface{} `json:"guid"`
		Type    string                 `json:"type"`
	}
	
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 处理事件数据
	var mintEvents []BridgeMintEvent
	for _, event := range events {
		var mintEvent BridgeMintEvent
		
		// 从event.Data中提取字段并正确转换类型
		if amountStr, ok := event.Data["amount"].(string); ok {
			// 将字符串转换为uint64
			amountBig, ok := new(big.Int).SetString(amountStr, 10)
			if !ok {
				return nil, fmt.Errorf("解析事件数据失败: 无法将amount转换为数字: %s", amountStr)
			}
			mintEvent.Amount = amountBig.Uint64()
		}
		
		if btcTxID, ok := event.Data["btc_tx_id"].(string); ok {
			mintEvent.BtcTxId = btcTxID
		}
		
		if receiver, ok := event.Data["receiver"].(string); ok {
			mintEvent.Receiver = receiver
		}
		
		mintEvents = append(mintEvents, mintEvent)
	}

	return mintEvents, nil
}
// GetRedeemRequestEvents 获取赎回请求事件
func GetRedeemRequestEvents(client *aptos.Client, moduleAddress string, limit uint64) ([]RedeemRequestEvent, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取BridgeEvents资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取桥事件资源失败: %v", err)
	}

	// 从资源数据中获取redeem_request_events
	data, ok := resource["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("资源数据格式不正确")
	}

	eventsData, ok := data["redeem_request_events"]
	if !ok {
		return nil, fmt.Errorf("资源中未找到redeem_request_events字段")
	}

	// 解析事件数据
	eventsMap, ok := eventsData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("redeem_request_events字段格式不正确")
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
		return []RedeemRequestEvent{}, nil
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

	// // 计算查询范围
	// start := uint64(0)
	// if counter.Uint64() > limit {
	// 	start = counter.Uint64() - limit
	// }

	// 使用硬编码的API URL
	baseURL := "https://fullnode.devnet.aptoslabs.com" // 默认使用devnet
	
	// 正确构建事件API路径
	// https://api.devnet.aptoslabs.com/v1/accounts/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d/events/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents/mint_events
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s/%s", 
		baseURL,
		accountAddress.String(),
		resourceType,
		"redeem_request_events")
	
	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	q := req.URL.Query()
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
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
	var events []struct {
		Version string                 `json:"version"`
		Key     string                 `json:"key"`
		Guid    map[string]interface{} `json:"guid"`
		Data    map[string]interface{} `json:"data"`
		SequenceNumber string          `json:"sequence_number"`
	}
	
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 处理事件数据
	var redeemEvents []RedeemRequestEvent
	for _, event := range events {
		var redeemEvent RedeemRequestEvent
		
		// 从event.Data中提取字段并正确转换类型
		if amountStr, ok := event.Data["amount"].(string); ok {
			// 将字符串转换为uint64
			amountBig, ok := new(big.Int).SetString(amountStr, 10)
			if !ok {
				return nil, fmt.Errorf("解析事件数据失败: 无法将amount转换为数字: %s", amountStr)
			}
			redeemEvent.Amount = amountBig.Uint64()
		}
		
		if sender, ok := event.Data["sender"].(string); ok {
			redeemEvent.Sender = sender
		}
		
		if receiver, ok := event.Data["receiver"].(string); ok {
			redeemEvent.Receiver = receiver
		}
		
		redeemEvents = append(redeemEvents, redeemEvent)
	}

	return redeemEvents, nil
}

// GetRedeemPrepareEvents 获取赎回准备事件
func GetRedeemPrepareEvents(client *aptos.Client, moduleAddress string, limit uint64) ([]RedeemPrepareEvent, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 使用硬编码的API URL
	baseURL := "https://fullnode.devnet.aptoslabs.com" // 默认使用devnet
	
	// 正确构建事件API路径
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	eventPath := fmt.Sprintf("%s/redeem_prepare_events", resourceType)
	
	// 构建完整的URL
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s", 
		baseURL,
		address.String(),
		eventPath)
	
	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	q := req.URL.Query()
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
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
	var events []struct {
		Version string                 `json:"version"`
		Key     string                 `json:"key"`
		Guid    map[string]interface{} `json:"guid"`
		Data    map[string]interface{} `json:"data"`
		SequenceNumber string          `json:"sequence_number"`
	}
	
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 处理事件数据
	var prepareEvents []RedeemPrepareEvent
	for _, event := range events {
		var prepareEvent RedeemPrepareEvent
		data, err := json.Marshal(event.Data)
		if err != nil {
			return nil, fmt.Errorf("序列化事件数据失败: %v", err)
		}

		err = json.Unmarshal(data, &prepareEvent)
		if err != nil {
			return nil, fmt.Errorf("解析事件数据失败: %v", err)
		}

		prepareEvents = append(prepareEvents, prepareEvent)
	}

	return prepareEvents, nil
}

// GetTokenMintEvents 获取代币铸造事件
func GetTokenMintEvents(client *aptos.Client, moduleAddress string, limit uint64) ([]TokenMintEvent, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 使用硬编码的API URL
	baseURL := "https://fullnode.devnet.aptoslabs.com" // 默认使用devnet
	
	// 正确构建事件API路径
	resourceType := fmt.Sprintf("%s::btc_tokenv3::BridgeEvents", address.String())
	eventPath := fmt.Sprintf("%s/mint_events", resourceType)
	
	// 构建完整的URL
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s", 
		baseURL,
		address.String(),
		eventPath)
	
	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	q := req.URL.Query()
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
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
	var events []struct {
		Version string                 `json:"version"`
		Key     string                 `json:"key"`
		Guid    map[string]interface{} `json:"guid"`
		Data    map[string]interface{} `json:"data"`
		SequenceNumber string          `json:"sequence_number"`
	}
	
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 处理事件数据
	var mintEvents []TokenMintEvent
	for _, event := range events {
		var mintEvent TokenMintEvent
		data, err := json.Marshal(event.Data)
		if err != nil {
			return nil, fmt.Errorf("序列化事件数据失败: %v", err)
		}

		err = json.Unmarshal(data, &mintEvent)
		if err != nil {
			return nil, fmt.Errorf("解析事件数据失败: %v", err)
		}

		mintEvents = append(mintEvents, mintEvent)
	}

	return mintEvents, nil
}

// GetTokenBurnEvents 获取代币燃烧事件
func GetTokenBurnEvents(client *aptos.Client, moduleAddress string, limit uint64) ([]TokenBurnEvent, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 使用硬编码的API URL
	baseURL := "https://fullnode.devnet.aptoslabs.com" // 默认使用devnet
	
	// 正确构建事件API路径
	resourceType := fmt.Sprintf("%s::btc_tokenv3::BridgeEvents", address.String())
	eventPath := fmt.Sprintf("%s/burn_events", resourceType)
	
	// 构建完整的URL
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s", 
		baseURL,
		address.String(),
		eventPath)
	
	// 添加查询参数
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	
	q := req.URL.Query()
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
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
	var events []struct {
		Version string                 `json:"version"`
		Key     string                 `json:"key"`
		Guid    map[string]interface{} `json:"guid"`
		Data    map[string]interface{} `json:"data"`
		SequenceNumber string          `json:"sequence_number"`
	}
	
	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	
	// 处理事件数据
	var burnEvents []TokenBurnEvent
	for _, event := range events {
		var burnEvent TokenBurnEvent
		data, err := json.Marshal(event.Data)
		if err != nil {
			return nil, fmt.Errorf("序列化事件数据失败: %v", err)
		}

		err = json.Unmarshal(data, &burnEvent)
		if err != nil {
			return nil, fmt.Errorf("解析事件数据失败: %v", err)
		}

		burnEvents = append(burnEvents, burnEvent)
	}

	return burnEvents, nil
}

// GetPreparedRedeems 获取已准备的赎回列表
func GetPreparedRedeems(client *aptos.Client, moduleAddress string) ([]string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取PreparedRedeems资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::PreparedRedeems", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取已准备赎回列表失败: %v", err)
	}

	// 修复: 直接访问map
	preparedData, ok := resource["prepared"]
	if !ok {
		return nil, fmt.Errorf("资源中未找到prepared字段")
	}

	// 解析prepared列表
	var prepared []string
	preparedList, ok := preparedData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("prepared字段格式不正确")
	}

	for _, tx := range preparedList {
		txStr, ok := tx.(string)
		if !ok {
			return nil, fmt.Errorf("交易ID格式不正确")
		}
		prepared = append(prepared, txStr)
	}

	return prepared, nil
}

// GetUsedBtcTxIds 获取已使用的BTC交易ID列表
func GetUsedBtcTxIds(client *aptos.Client, moduleAddress string) ([]string, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	// 获取UsedBtcTxIds资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::UsedBtcTxIds", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取已使用交易ID列表失败: %v", err)
	}

	// 修复: 直接访问map
	usedData, ok := resource["used"]
	if !ok {
		return nil, fmt.Errorf("资源中未找到used字段")
	}

	// 解析used列表
	var used []string
	usedList, ok := usedData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("used字段格式不正确")
	}

	for _, tx := range usedList {
		txStr, ok := tx.(string)
		if !ok {
			return nil, fmt.Errorf("交易ID格式不正确")
		}
		used = append(used, txStr)
	}

	return used, nil
}

// QueryBridgeStatus 查询桥的状态信息
func QueryBridgeStatus(client *aptos.Client, moduleAddress string, checkLoopTime int) error {
	// 查询桥配置
	config, err := GetBridgeConfig(client, moduleAddress)
	if err != nil {
		return fmt.Errorf("获取桥配置失败: %v", err)
	}
	
	// 打印配置信息
	fmt.Println("===== 桥配置信息 =====")
	// fmt.Println(config) // map[data:map[admin:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d fee:2 fee_account:0xa2184ca989d7741a463c0a77f0283aab71c86410f6596a814b217329573488ee] type:0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeConfig]
	adminStr := config["data"].(map[string]interface{})["admin"].(string)
	feeStr := config["data"].(map[string]interface{})["fee"].(string)
	feeAccountStr := config["data"].(map[string]interface{})["fee_account"].(string)
	
	fmt.Printf("管理员地址: %s\n", adminStr)
	fee, _ := new(big.Int).SetString(feeStr, 10)
	fmt.Printf("交易费用: %d (satoshi)\n", fee)
	fmt.Printf("费用接收地址: %s\n", feeAccountStr)
	for {
		// 显示当前查询时间
		currentTime := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("\n===== 查询时间: %s =====\n", currentTime)
		
		// 查询最近的铸币事件
		mintEvents, err := GetBridgeMintEvents(client, moduleAddress, 5)
		if err != nil {
			fmt.Printf("获取铸币事件失败: %v\n", err)
		} else {
			fmt.Println("\n===== 最近5个铸币事件 =====")
			if len(mintEvents) == 0 {
				fmt.Println("暂无铸币事件")
			}
			for i, event := range mintEvents {
				fmt.Printf("%d. 交易ID: %s, 接收者: %s, 金额: %d\n", 
					i+1, event.BtcTxId, event.Receiver, event.Amount)
			}
		}
		
		// 查询最近的赎回请求事件
		redeemEvents, err := GetRedeemRequestEvents(client, moduleAddress, 5)
		if err != nil {
			fmt.Printf("获取赎回请求事件失败: %v\n", err)
		} else {
			fmt.Println("\n===== 最近5个赎回请求事件 =====")
			if len(redeemEvents) == 0 {
				fmt.Println("暂无赎回请求事件")
			}
			for i, event := range redeemEvents {
				fmt.Printf("%d. 发送者: %s, 接收者: %s, 金额: %d\n", 
					i+1, event.Sender, event.Receiver, event.Amount)
			}
		}
		
		fmt.Printf("\n等待 %d 秒后进行下一次查询...\n", checkLoopTime)
		time.Sleep(time.Duration(checkLoopTime) * time.Second)
	}

	
	return nil
}


