package etherman

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	mybridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/TEENet-io/bridge-go/multisig_client"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	logger "github.com/sirupsen/logrus"
)

// This file contains a REAL ethereum chain envinorment for the bridge.
// that are used to interact with a real ETH network.
// You can use these in tests, or in real applications.

// Fully terraformed ethereum chain for bridge.
type RealEthChain struct {
	RpcClient             *ethclient.Client             // work with ethereum chain
	ChainId               *big.Int                      // chain id of the ethereum network
	CoreAccount           *bind.TransactOpts            // bridge controlled account (with money) to sign Mint(), Prepare() transactions
	SchnorrSigner         multisig_client.SchnorrSigner // Pub() key to be fed into smart contract.
	BridgeContractAddress common.Address                // created during New process
	TwbtcContractAddress  common.Address                // created during New process
}

// rpc_url: the ethereum json rpc to connect to.
// priv_key: the private key of the bridge controlled account.
// schnorrSigner: the schnorr signer to be used in the bridge contract.
// predefinedBridgeAddress: if you have a pre-deployed bridge contract, provide its address here, or "" if you haven't.
// predefinedTwbtcAddress: if you have a pre-deployed twbtc contract, provide its address here, or "" if you haven't.
func NewRealEthChain(
	rpc_url string,
	priv_key *ecdsa.PrivateKey,
	schnorrSigner multisig_client.SchnorrSigner,
	predefinedBridgeAddress string,
	predefinedTwbtcAddress string,
) (*RealEthChain, error) {
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
	coreAccount.GasLimit = 728410 // set a higher gas limit to prevent deploy bridge contract fail

	var bridgeAddress common.Address
	var contract *mybridge.TEENetBtcBridge

	// Deploy bridge smart contracts (if pre-deployed then simply bind to them).
	if predefinedBridgeAddress != "" && predefinedTwbtcAddress != "" {
		logger.Info("Using predefined bridge and twbtc contracts")
		bridgeAddress = common.HexToAddress(predefinedBridgeAddress)
		// check the exist of smart contract on this address
		code, err := client.CodeAt(context.Background(), bridgeAddress, nil)
		if err != nil {
			logger.Fatalf("[evm] Failed to get code at address: %s %v", predefinedBridgeAddress, err)
			return nil, err
		}

		if len(code) == 0 {
			logger.Errorf("[evm] No smart contract is deployed at this address. %s", predefinedBridgeAddress)
			return nil, fmt.Errorf("[evm] address %s doesn't contain smart contract", predefinedBridgeAddress)
		}
	} else {
		logger.Info("Deploying new bridge and twbtc contracts")
		pub_key, err := schnorrSigner.Pub()
		if err != nil {
			return nil, err
		}
		pk_x, _ := multisig_client.BtcEcPubKeyToXY(pub_key)
		logger.Info("Schnorr pub key received")

		var deployTx *types.Transaction
		bridgeAddress, deployTx, contract, err = mybridge.DeployTEENetBtcBridge(
			coreAccount,
			client,
			pk_x)
		if err != nil {
			return nil, err
		}
		logger.WithFields(logger.Fields{
			"bridge_addr": bridgeAddress.Hex(),
			"deploy_tx":   deployTx.Hash().Hex(),
		}).Info("Bridge contract deploy Tx sent")

		// Wait for the deployment transaction to be mined
		_, err = bind.WaitDeployed(context.Background(), client, deployTx)
		if err != nil {
			logger.Fatalf("Deployment tx not mined or failed: %v", err)
			return nil, err
		}
		logger.Info("Deploy Complete")

		// Compare bridge contract public key with the one we provided
		_pk, err := contract.Pk(nil)
		if err != nil {
			return nil, err
		}
		if pk_x.Cmp(_pk) != 0 {
			return nil, err
		}
	}

	// Binding!
	// bridgeContract is a functional golang entity of the deployed bridge contract
	bridgeContract, err := mybridge.NewTEENetBtcBridge(bridgeAddress, client)
	if err != nil {
		return nil, err
	}

	// TWBTC contract address (inferrable from bridge contract)
	twbtcAddress, err := bridgeContract.Twbtc(nil)
	if err != nil {
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
