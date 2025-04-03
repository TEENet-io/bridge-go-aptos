package aptosman

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"

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
			Args:     [][]byte{},
		},
	},
	)
	if err != nil {
		return "", fmt.Errorf("构建交易失败: %v", err)
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
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
		return "", fmt.Errorf("构建交易失败: %v", err)
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		return "", fmt.Errorf("提交交易失败: %v", err)
	}
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		return "", fmt.Errorf("等待交易确认失败: %v", err)
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

// func registerTWBTC(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress aptos.AccountAddress) (string, error) {
// 	address := aptos.AccountAddress{}
// 	err := address.ParseStringRelaxed(moduleAddress)
// 	if err != nil {
// 		return "", fmt.Errorf("解析地址失败: %v", err)
// 	}

// 	receiverAddressBytes, err := bcs.Serialize(&receiverAddress)
// 	if err != nil {
// 		return "", fmt.Errorf("序列化接收方地址失败: %v", err)
// 	}

// 	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
// 		Payload: &aptos.EntryFunction{
// 			Module: aptos.ModuleId{
// 				Address: address,
// 				Name:    "btc_tokenv3",
// 			},
// 			Function: "registerv2",
// 			ArgTypes: []aptos.TypeTag{},
// 			Args: [][]byte{receiverAddressBytes},
// 		},
// 	},
// 	)

// 	if err != nil {
// 		panic("Failed to build transaction:" + err.Error())
// 	}
// 	signedTxn, err := rawTxn.SignedTransaction(account)
// 	if err != nil {
// 		panic("Failed to sign transaction:" + err.Error())
// 	}

// 	submitResult, err := client.SubmitTransaction(signedTxn)
// 	if err != nil {
// 		panic("Failed to submit transaction:" + err.Error())
// 	}
// 	txnHash := submitResult.Hash

// 	_, err = client.WaitForTransaction(txnHash)
// 	if err != nil {
// 		panic("Failed to wait for transaction:" + err.Error())
// 	}

// 	// 验证交易是否成功
// 	txnInfo, err := client.TransactionByHash(txnHash)
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

func mintTWBTC(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress aptos.AccountAddress, amount uint64, btc_tx_id string) (MintParams, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return MintParams{}, fmt.Errorf("解析地址失败: %v", err)
	}

	// btc_tx_id: String,
	// receiver: address,
	// amount: u64,

	btc_tx_id_len := len(btc_tx_id)
	btc_tx_id_bytes := make([]byte, 0)

	if btc_tx_id_len < 128 {
		btc_tx_id_bytes = append(btc_tx_id_bytes, byte(btc_tx_id_len))
	} else {
		encodedLen := byte(btc_tx_id_len) | 0x80
		btc_tx_id_bytes = append(btc_tx_id_bytes, encodedLen, byte(btc_tx_id_len>>7))
	}

	btc_tx_id_bytes = append(btc_tx_id_bytes, []byte(btc_tx_id)...)

	receiverAddressBytes, err := bcs.Serialize(&receiverAddress)
	if err != nil {
		return MintParams{}, fmt.Errorf("序列化接收方地址失败: %v", err)
	}
	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return MintParams{}, fmt.Errorf("序列化金额失败: %v", err)
	}

	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_bridgev3",
			},
			Function: "mint",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{btc_tx_id_bytes, receiverAddressBytes, amountBytes},
		},
	},
	)
	if err != nil {
		return MintParams{}, fmt.Errorf("构建交易失败: %v", err)
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return MintParams{}, fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		return MintParams{}, fmt.Errorf("提交交易失败: %v", err)
	}
	txnHash := submitResult.Hash
	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		return MintParams{}, fmt.Errorf("等待交易确认失败: %v", err)
	}

	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return MintParams{}, fmt.Errorf("获取交易信息失败: %v", err)
	}
	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return MintParams{}, fmt.Errorf("解析用户交易信息失败: %v", err)
	}
	if userTxn.Success {
		var mintParams MintParams
		mintParams.BtcTxId = btc_tx_id_bytes
		mintParams.Amount = amount
		mintParams.Receiver = receiverAddress.String()
		return mintParams, nil
	} else {
		return MintParams{}, fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}

}

func string_to_bytes(str string) []byte {
	len := len(str)
	bytes := make([]byte, 0)
	if len < 128 {
		bytes = append(bytes, byte(len))
	} else {
		encodedLen := byte(len) | 0x80
		bytes = append(bytes, encodedLen, byte(len>>7))
	}
	bytes = append(bytes, []byte(str)...)

	return bytes
}

// 修改 redeemPrepare 函数签名，使其接受 [][32]byte 和 []uint16 类型参数
func redeemPrepare(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string,
	redeem_request_tx_hash string, requester string, receiverAddress string,
	amount uint64, outpointTxIds [][32]byte, outpointIdxs []uint16) (PrepareParams, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("解析模块地址失败: %v", err)
	}

	txHashBytes := serializeString(redeem_request_tx_hash)

	requesterAddr := aptos.AccountAddress{}
	err = requesterAddr.ParseStringRelaxed(requester)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("解析requester地址失败: %v", err)
	}
	requesterBytes, err := bcs.Serialize(&requesterAddr)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("序列化requester地址失败: %v", err)
	}

	receiverBytes := serializeString(receiverAddress)

	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("序列化金额失败: %v", err)
	}

	stringTxIds := make([]string, len(outpointTxIds))
	for i, txid := range outpointTxIds {
		stringTxIds[i] = hex.EncodeToString(txid[:])
	}
	outpointTxIdsBytes := serializeStringVector(stringTxIds)

	uint64Idxs := make([]uint64, len(outpointIdxs))
	for i, idx := range outpointIdxs {
		uint64Idxs[i] = uint64(idx)
	}
	outpointIdxsBytes := serializeU64Vector(uint64Idxs)

	// 构建交易
	rawTxn, err := client.BuildTransaction(account.AccountAddress(), aptos.TransactionPayload{
		Payload: &aptos.EntryFunction{
			Module: aptos.ModuleId{
				Address: address,
				Name:    "btc_bridgev3",
			},
			Function: "redeem_prepare",
			ArgTypes: []aptos.TypeTag{},
			Args:     [][]byte{txHashBytes, requesterBytes, receiverBytes, amountBytes, outpointTxIdsBytes, outpointIdxsBytes},
		},
	})

	if err != nil {
		return PrepareParams{}, fmt.Errorf("构建交易失败: %v", err)
	}

	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("签名交易失败: %v", err)
	}

	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("提交交易失败: %v", err)
	}

	// 不需要验证交易是否成功，直接返回
	txnHash := submitResult.Hash

	_, err = client.WaitForTransaction(txnHash)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("等待交易确认失败: %v", err)
	}

	// 验证交易是否成功
	txnInfo, err := client.TransactionByHash(txnHash)
	if err != nil {
		return PrepareParams{}, fmt.Errorf("获取交易信息失败: %v", err)
	}

	userTxn, err := txnInfo.UserTransaction()
	if err != nil {
		return PrepareParams{}, fmt.Errorf("解析用户交易信息失败: %v", err)
	}

	if userTxn.Success {
		var prepareParams PrepareParams
		prepareParams.RequestTxHash = redeem_request_tx_hash
		prepareParams.Requester = requester
		prepareParams.Receiver = receiverAddress
		prepareParams.Amount = amount
		prepareParams.OutpointTxIds = stringTxIds
		prepareParams.OutpointIdxs = outpointIdxs
		return prepareParams, nil
	} else {
		return PrepareParams{}, fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	}
}

func serializeString(s string) []byte {
	strLen := len(s)
	result := make([]byte, 0, strLen+1)

	if strLen < 128 {
		result = append(result, byte(strLen))
	} else {
		result = append(result, byte(strLen&0x7F|0x80), byte(strLen>>7))
	}

	result = append(result, []byte(s)...)
	return result
}

func serializeStringVector(strings []string) []byte {
	result := make([]byte, 0)

	vecLen := len(strings)
	if vecLen < 128 {
		result = append(result, byte(vecLen))
	} else {
		result = append(result, byte(vecLen&0x7F|0x80), byte(vecLen>>7))
	}

	// 序列化每个字符串
	for _, s := range strings {
		result = append(result, serializeString(s)...)
	}

	return result
}

func serializeU64Vector(nums []uint64) []byte {
	result := make([]byte, 0)

	vecLen := len(nums)
	if vecLen < 128 {
		result = append(result, byte(vecLen))
	} else {
		result = append(result, byte(vecLen&0x7F|0x80), byte(vecLen>>7))
	}

	for _, num := range nums {
		numBytes, _ := bcs.SerializeU64(num)
		result = append(result, numBytes...)
	}

	return result
}

func redeemRequest(client *aptos.Client, account aptos.TransactionSigner, moduleAddress string, receiverAddress string, amount uint64) (RequestParams, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return RequestParams{}, fmt.Errorf("解析地址失败: %v", err)
	}
	amountBytes, err := bcs.SerializeU64(amount)
	if err != nil {
		return RequestParams{}, fmt.Errorf("序列化金额失败: %v", err)
	}

	receiverStrLen := len(receiverAddress)
	receiverBytes := make([]byte, 0)

	if receiverStrLen < 128 {
		receiverBytes = append(receiverBytes, byte(receiverStrLen))
	} else {

		encodedLen := byte(receiverStrLen) | 0x80
		receiverBytes = append(receiverBytes, encodedLen, byte(receiverStrLen>>7))
	}

	receiverBytes = append(receiverBytes, []byte(receiverAddress)...)

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
		return RequestParams{}, fmt.Errorf("构建交易失败: %v", err)
	}
	signedTxn, err := rawTxn.SignedTransaction(account)
	if err != nil {
		return RequestParams{}, fmt.Errorf("签名交易失败: %v", err)
	}
	_, err = client.SubmitTransaction(signedTxn)
	if err != nil {
		return RequestParams{}, fmt.Errorf("提交交易失败: %v", err)
	}

	redeemRequestParams := RequestParams{
		Amount:   amount,
		Receiver: receiverAddress,
	}

	// _, err = client.WaitForTransaction(txnHash)
	// if err != nil {
	// 	panic("Failed to wait for transaction:" + err.Error())
	// }
	// // 验证交易是否成功
	// txnInfo, err := client.TransactionByHash(txnHash)
	// if err != nil {
	// 	return nil, fmt.Errorf("获取交易信息失败: %v", err)
	// }
	// userTxn, err := txnInfo.UserTransaction()
	// if err != nil {
	// 	return nil, fmt.Errorf("解析用户交易信息失败: %v", err)
	// }
	// if userTxn.Success {
	// 	return txnHash, nil
	// } else {
	// 	return nil, fmt.Errorf("交易执行失败: %s", userTxn.VmStatus)
	// }

	return redeemRequestParams, nil
}

// GetBridgeConfig 获取桥的配置信息
func GetBridgeConfig(client *aptos.Client, moduleAddress string) (map[string]interface{}, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeConfig", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取桥配置失败: %v", err)
	}

	return resource, nil
}

func GetBridgeEvents(client *aptos.Client, moduleAddress string, eventType string) (map[string]interface{}, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("解析模块地址失败: %v", err)
	}

	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	resource, err := client.AccountResource(address, resourceType)
	if err != nil {
		return nil, fmt.Errorf("获取桥事件资源失败: %v", err)
	}

	eventsData, ok := resource["data"].(map[string]interface{})[eventType]
	if !ok {
		return nil, fmt.Errorf("资源中未找到%s字段", eventType)
	}

	eventsMap, ok := eventsData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("资源数据格式不正确")
	}

	return eventsMap, nil
}

// https://api.devnet.aptoslabs.com/v1/accounts/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d/events/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents/redeem_request_events?start=1
// GetEvents 获取事件
func GetEvents(client *aptos.Client, moduleAddress string, events_field string, limit uint64, start uint64, aptos_url string) ([]map[string]interface{}, error) {
	address := aptos.AccountAddress{}
	err := address.ParseStringRelaxed(moduleAddress)
	if err != nil {
		return nil, fmt.Errorf("Parse module address failed: %v", err)
	}

	// 获取BridgeEvents资源
	resourceType := fmt.Sprintf("%s::btc_bridgev3::BridgeEvents", address.String())
	resource, err := client.AccountResource(address, resourceType)
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

	baseURL := aptos_url

	// 正确构建事件API路径
	fullURL := fmt.Sprintf("%s/v1/accounts/%s/events/%s/%s",
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
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	if start > 0 {
		q.Add("start", fmt.Sprintf("%d", start))
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
	var events []map[string]interface{}

	err = json.Unmarshal(body, &events)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return events, nil
}

// https://api.devnet.aptoslabs.com/v1/accounts/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d/events/0x1319db9743efbef92e2ed32e122a4690f466fbbb8e34cd6ccffb93e8cb68447d::btc_bridgev3::BridgeEvents/redeem_prepare_events?limit=1&start=0
// GetRedeemPreparedEvents 获取赎回准备事件
// // 赎回准备事件
//
//	type RedeemPreparedEvent struct {
//		TxVersion     string     // Aptos交易版本号
//		EthTxHash     string     // 请求交易哈希
//		Requester     string     // 请求者地址
//		Receiver      string     // 接收者比特币地址
//		Amount        uint64     // 金额
//		OutpointTxIds []string   // 比特币UTXO交易ID列表
//		OutpointIdxs  []uint16   // 对应的输出索引
//	}
func (aptman *Aptosman) GetRedeemPreparedEvents(client *aptos.Client, moduleAddress string, limit uint64, start uint64) ([]RedeemPreparedEvent, error) {
	baseURL := aptman.cfg.URL
	events, err := GetEvents(client, moduleAddress, "redeem_prepare_events", limit, start, baseURL)
	if err != nil {
		return nil, err
	}

	type EventResponse struct {
		Version string `json:"version"`
		Guid    struct {
			CreationNumber string `json:"creation_number"`
			AccountAddress string `json:"account_address"`
		} `json:"guid"`
		SequenceNumber string `json:"sequence_number"`
		Type           string `json:"type"`
		Data           struct {
			Amount        string   `json:"amount"`
			EthTxHash     string   `json:"eth_tx_hash"`
			OutpointIdxs  []string `json:"outpoint_idxs"`
			OutpointTxIds []string `json:"outpoint_tx_ids"`
			Receiver      string   `json:"receiver"`
			Requester     string   `json:"requester"`
		} `json:"data"`
	}

	var prepareEvents []RedeemPreparedEvent
	for _, event := range events {
		// 将map转换为JSON
		eventData, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("serialize event data failed: %v", err)
		}

		// 解析为临时结构体
		var eventResponse EventResponse
		if err := json.Unmarshal(eventData, &eventResponse); err != nil {
			return nil, fmt.Errorf("parse event data failed: %v", err)
		}

		// 将数据转换为RedeemPreparedEvent
		var prepareEvent RedeemPreparedEvent
		prepareEvent.RequestTxHash = eventResponse.Version
		prepareEvent.PrepareTxHash = eventResponse.Data.EthTxHash
		prepareEvent.Requester = eventResponse.Data.Requester
		prepareEvent.Receiver = eventResponse.Data.Receiver
		prepareEvent.OutpointTxIds = eventResponse.Data.OutpointTxIds
		// 将amount从字符串转换为uint64
		amount, err := strconv.ParseUint(eventResponse.Data.Amount, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse amount failed: %v", err)
		}
		prepareEvent.Amount = amount

		// 将outpoint_idxs从字符串数组转换为uint16数组
		outpointIdxs := make([]uint16, len(eventResponse.Data.OutpointIdxs))
		for i, idxStr := range eventResponse.Data.OutpointIdxs {
			idx, err := strconv.ParseUint(idxStr, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("parse outpoint_idxs failed: %v", err)
			}
			outpointIdxs[i] = uint16(idx)
		}
		prepareEvent.OutpointIdxs = outpointIdxs

		prepareEvents = append(prepareEvents, prepareEvent)
	}

	return prepareEvents, nil
}

// GetRedeemRequestEvents 获取赎回请求事件
// 赎回请求事件
//
//	type RedeemRequestEvent struct {
//		TxVersion    string     // Aptos交易版本号
//		Sender       string     // 发送者Aptos地址
//		Receiver     string     // 接收者比特币地址
//		Amount       uint64     // 金额
//	}
func (aptman *Aptosman) GetRedeemRequestedEvents(client *aptos.Client, moduleAddress string, limit uint64, start uint64) ([]RedeemRequestedEvent, error) {
	baseURL := aptman.cfg.URL
	events, err := GetEvents(client, moduleAddress, "redeem_request_events", limit, start, baseURL)
	if err != nil {
		return nil, err
	}

	type EventResponse struct {
		Version string `json:"version"`
		Guid    struct {
			CreationNumber string `json:"creation_number"`
			AccountAddress string `json:"account_address"`
		} `json:"guid"`
		SequenceNumber string `json:"sequence_number"`
		Type           string `json:"type"`
		Data           struct {
			Sender   string `json:"sender"`
			Amount   string `json:"amount"`
			Receiver string `json:"receiver"`
		} `json:"data"`
	}

	var redeemEvents []RedeemRequestedEvent
	for _, event := range events {
		// 将map转换为JSON
		eventData, err := json.Marshal(event)
		if err != nil {
			return nil, fmt.Errorf("serialize event data failed: %v", err)
		}

		// 解析为临时结构体
		var eventResponse EventResponse
		if err := json.Unmarshal(eventData, &eventResponse); err != nil {
			return nil, fmt.Errorf("parse event data failed: %v", err)
		}

		// 将数据转换为RedeemRequestEvent
		var redeemEvent RedeemRequestedEvent
		redeemEvent.RequestTxHash = eventResponse.Version
		redeemEvent.Requester = eventResponse.Data.Sender
		redeemEvent.Receiver = eventResponse.Data.Receiver

		// 将amount从字符串转换为uint64
		amount, err := strconv.ParseUint(eventResponse.Data.Amount, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse amount failed: %v", err)
		}
		redeemEvent.Amount = amount

		redeemEvents = append(redeemEvents, redeemEvent)
	}

	// fmt.Println("redeemEvents", redeemEvents)
	return redeemEvents, nil
}
