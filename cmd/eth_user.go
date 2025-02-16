package cmd

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/etherman"
)

// EthUser's configuration
type EthUserConfig struct {
	EthRpcUrl          string // json rpc url
	EthCoreAccountPriv string // private key of the user controlled account
	EthBridgeAddress   string // address of the bridge contract
	EthTwbtcAddress    string // address of the twbtc contract
}

// EthUser is a user of the bridge on eth-side.
type EthUser struct {
	MyRpcClient           *ethclient.Client  // connect with ethereum chain via rpc.
	MyEtherman            *etherman.Etherman // Call the smart contract methods over rpc.
	ChainId               *big.Int           // chain id of the ethereum network (gained upon rpc connection)
	CoreAccount           *bind.TransactOpts // bridge controlled account (with money) to sign Mint(), Prepare() transactions
	BridgeContractAddress common.Address     // created during New process
	TwbtcContractAddress  common.Address     // created during New process
}

// Create a new EthUser object.
func NewEthUser(euc *EthUserConfig) (*EthUser, error) {
	myRpcClient, err := ethclient.Dial(euc.EthRpcUrl)
	if err != nil {
		logger.Fatalf("Failed to connect to the Ethereum client: %v", err)
		return nil, err
	}

	// Check Chain ID from rpc!
	chainId, err := myRpcClient.ChainID(context.Background())
	if err != nil {
		logger.Fatalf("Failed to get chain id via rpc: %v", err)
		return nil, err
	}

	// Create auth object from private key
	_priv, err := etherman.StringToPrivateKey(euc.EthCoreAccountPriv)
	if err != nil {
		logger.Fatalf("Failed to convert private key to ecdsa private key: %v", err)
		return nil, err
	}
	coreAccount := etherman.NewAuth(_priv, chainId)

	// Convert bridge address to common.Address
	bridgeAddress := common.HexToAddress(euc.EthBridgeAddress)
	twbtcAddress := common.HexToAddress(euc.EthTwbtcAddress)

	// Create the Etherman Config
	ec := &etherman.EthermanConfig{
		URL:                   euc.EthRpcUrl,
		BridgeContractAddress: bridgeAddress,
		TWBTCContractAddress:  twbtcAddress,
	}
	// Create the Etherman object
	myEtherman, err := etherman.NewEtherman(ec, coreAccount)
	if err != nil {
		logger.Fatalf("Failed to create Etherman object: %v", err)
		return nil, err
	}

	return &EthUser{
		MyRpcClient:           myRpcClient,
		MyEtherman:            myEtherman,
		ChainId:               chainId,
		CoreAccount:           coreAccount,
		BridgeContractAddress: bridgeAddress,
		TwbtcContractAddress:  twbtcAddress,
	}, nil
}

// Close and relese resources.
func (eu *EthUser) Close() {
	eu.MyRpcClient.Close()
}

// Fetch the user's address.
func (eu *EthUser) GetAddress() string {
	return eu.CoreAccount.From.Hex()
}

// Fetch the eth balance of the user's account.
func (eu *EthUser) GetBalance() (*big.Int, error) {
	return eu.MyEtherman.GetBalance(eu.CoreAccount.From)
}

// Fetch the TWBTC balance of the user's account.
func (eu *EthUser) GetTwbtcBalance() (*big.Int, error) {
	return eu.MyEtherman.TWBTCBalanceOf(eu.CoreAccount.From)
}

// amount: amount of TWBTC to redeem
// btcReceiver: the receiver's btc address
func (eu *EthUser) InitiateRedeem(amount *big.Int, btcReceiver string) (*common.Hash, error) {
	// Safe guard
	_twbtc, err := eu.MyEtherman.TWBTCBalanceOf(eu.CoreAccount.From)
	if err != nil {
		return nil, err
	}
	if _twbtc.Cmp(amount) < 0 {
		return nil, fmt.Errorf("not enough TWBTC balance: have %s, need %s", _twbtc.String(), amount.String())
	}
	// Approve the bridge to use the user's TWBTC
	_approve_tx, err := eu.MyEtherman.TWBTCApprove(eu.CoreAccount, amount)
	if err != nil {
		return nil, err
	}
	// Wait for approve
	_approve_success, err := eu.MyEtherman.WaitForTxReceipt(_approve_tx, 20, 6)
	if err != nil {
		return nil, err
	}
	if !_approve_success {
		return nil, fmt.Errorf("failed to approve TWBTC transfer TxID %s, amount %d", _approve_tx.Hash().Hex(), amount)
	}

	// prepare redeem params
	rp := &etherman.RequestParams{
		Amount:   amount,
		Receiver: btcReceiver,
	}
	// send redeem request
	_redeem_request_tx, err := eu.MyEtherman.RedeemRequest(eu.CoreAccount, rp)
	if err != nil {
		return nil, err
	}
	// Wait
	_redeem_request_success, err := eu.MyEtherman.WaitForTxReceipt(_redeem_request_tx, 20, 6)
	if err != nil {
		return nil, err
	}
	if !_redeem_request_success {
		return nil, fmt.Errorf("failed to send Redeem Request TxID %s, amount %d, btc_receiver %s", _redeem_request_tx.Hash().Hex(), amount, btcReceiver)
	}
	hash := _redeem_request_tx.Hash()
	return &hash, nil
}
