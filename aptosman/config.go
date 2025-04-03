package aptosman

import (
	"github.com/aptos-labs/aptos-go-sdk"
	"github.com/btcsuite/btcd/chaincfg"
)

// AptosmanConfig 定义Aptos管理器的配置参数
type AptosmanConfig struct {
	// Aptos节点URL
	URL string

	// 模块发布者地址（合约所有者）
	ModuleAddress string

	// 比特币网络配置
	BtcChainConfig *chaincfg.Params

	// 网络类型: mainnet, testnet, devnet
	Network string
}

// 定义网络类型常量
const (
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"
	NetworkDevnet  = "devnet"
)

// 获取适当的Aptos网络配置
func GetNetworkConfig(network string) (string, aptos.NetworkConfig) {
	switch network {
	case NetworkMainnet:
		url := "https://fullnode.mainnet.aptoslabs.com/v1"
		networkConfig := aptos.MainnetConfig
		return url, networkConfig
	case NetworkTestnet:
		url := "https://fullnode.testnet.aptoslabs.com/v1"
		networkConfig := aptos.TestnetConfig
		return url, networkConfig
	case NetworkDevnet:
		url := "https://fullnode.devnet.aptoslabs.com/v1"
		networkConfig := aptos.DevnetConfig
		return url, networkConfig
	default:
		url := "https://fullnode.devnet.aptoslabs.com/v1"
		networkConfig := aptos.DevnetConfig
		return url, networkConfig
	}
}
