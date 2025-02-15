package cmd

import (
	"os"

	btcrpc "github.com/TEENet-io/bridge-go/btcman/rpc"
	logger "github.com/sirupsen/logrus"
)

// fileExists checks if a file exists and is readable
func FileExists(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()
	return true
}

// Shared Helper function. Create a btc rpc client.
func SetupBtcRpc(server string, port string, username string, password string) (*btcrpc.RpcClient, error) {
	_config := btcrpc.RpcClientConfig{
		ServerAddr: server,
		Port:       port,
		Username:   username,
		Pwd:        password,
	}
	r, err := btcrpc.NewRpcClient(&_config)
	if err != nil {
		logger.Fatalf("failed to create btc rpc client: %v", err)
		return nil, err
	}
	return r, nil
}
