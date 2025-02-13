package etherman

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	mybridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/TEENet-io/bridge-go/multisig"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	logger "github.com/sirupsen/logrus"
)

// This file contains a REAL ethereum chain envinorment for the bridge.
// that are used to interact with a real ETH network.
// You can use these in tests, or in real applications.

// Fully terraformed ethereum chain for bridge.
type RealEthChain struct {
	RpcClient             *ethclient.Client      // work with ethereum chain
	ChainId               *big.Int               // chain id of the ethereum network
	CoreAccount           *bind.TransactOpts     // bridge controlled account (with money) to sign Mint(), Prepare() transactions
	SchnorrSigner         multisig.SchnorrSigner // Pub() key to be fed into smart contract.
	BridgeContractAddress common.Address         // created during New process
	TwbtcContractAddress  common.Address         // created during New process
}

func NewRealEthChain(rpc_url string, priv_key *ecdsa.PrivateKey, schnorrSigner multisig.SchnorrSigner) (*RealEthChain, error) {
	client, err := ethclient.Dial(rpc_url)
	if err != nil {
		logger.Fatalf("Failed to connect to the Ethereum client: %v", err)
		return nil, err
	}

	// Check Chain ID
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		logger.Fatalf("Failed to get chain id: %v", err)
		return nil, err
	}

	// It is pointless to check if bridge controlled eth account is of > 0 balance.
	// We can charge it later in run-time.
	// Skip!

	// Create auth object from private key
	coreAccount := NewAuth(priv_key, chainId)

	// Deploy bridge smart contracts.
	pk_x, _, err := schnorrSigner.Pub()
	if err != nil {
		return nil, err
	}

	bridgeAddress, deployTx, contract, err := mybridge.DeployTEENetBtcBridge(
		coreAccount,
		client,
		pk_x)
	if err != nil {
		return nil, err
	}

	// Wait for the deployment transaction to be mined
	_, err = bind.WaitDeployed(context.Background(), client, deployTx)
	if err != nil {
		logger.Fatalf("Deployment tx not mined or failed: %v", err)
		return nil, err
	}

	// bridgeContract is a functional golang entity of the deployed bridge contract
	bridgeContract, err := mybridge.NewTEENetBtcBridge(bridgeAddress, client)
	if err != nil {
		return nil, err
	}

	// TWBTC contract address
	twbtcAddress, err := bridgeContract.Twbtc(nil)
	if err != nil {
		return nil, err
	}

	// Compare bridge contract public key with the one we provided
	_pk, err := contract.Pk(nil)
	if err != nil {
		return nil, err
	}
	if pk_x.Cmp(_pk) != 0 {
		return nil, err
	}

	return &RealEthChain{
		RpcClient:             client,
		ChainId:               chainId,
		CoreAccount:           coreAccount,
		SchnorrSigner:         schnorrSigner,
		BridgeContractAddress: bridgeAddress,
		TwbtcContractAddress:  twbtcAddress,
	}, nil
}

// Fully terraformed ethereum chain for user.
type RealEthUserChain struct {
	RpcClient             *ethclient.Client  // work with ethereum chain
	ChainId               *big.Int           // chain id of the ethereum network
	UserAccount           *bind.TransactOpts // user controlled account (with money) to sign Request() transaction.
	BridgeContractAddress common.Address     // send Request() to this contract
	TwbtcContractAddress  common.Address     // send Approve(), BalanceOf() to this contract
}

func NewRealEthUserChain(rpc_url string, priv_key *ecdsa.PrivateKey, bridgeAddr common.Address, twbtcAddr common.Address) (*RealEthUserChain, error) {
	client, err := ethclient.Dial(rpc_url)
	if err != nil {
		logger.Fatalf("Failed to connect to the Ethereum client: %v", err)
		return nil, err
	}

	// Check Chain ID
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		logger.Fatalf("Failed to get chain id: %v", err)
		return nil, err
	}

	// Create auth object from private key
	userAccount := NewAuth(priv_key, chainId)

	return &RealEthUserChain{
		RpcClient:             client,
		ChainId:               chainId,
		UserAccount:           userAccount,
		BridgeContractAddress: bridgeAddr,
		TwbtcContractAddress:  twbtcAddr,
	}, nil
}
