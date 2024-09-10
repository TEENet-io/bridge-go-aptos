// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package TWBTC

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// TWBTCMetaData contains all meta data concerning the TWBTC contract.
var TWBTCMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner_\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"allowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"needed\",\"type\":\"uint256\"}],\"name\":\"ERC20InsufficientAllowance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"needed\",\"type\":\"uint256\"}],\"name\":\"ERC20InsufficientBalance\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"approver\",\"type\":\"address\"}],\"name\":\"ERC20InvalidApprover\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"receiver\",\"type\":\"address\"}],\"name\":\"ERC20InvalidReceiver\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"ERC20InvalidSender\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"ERC20InvalidSpender\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"currentSupply\",\"type\":\"uint256\"}],\"name\":\"TotalMintedExceedsMaxSupply\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"MAX_BTC_SUPPLY\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"burnFrom\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60806040523480156200001157600080fd5b50604051620019c2380380620019c2833981810160405281019062000037919062000288565b806040518060400160405280601681526020017f5445454e6574207772617070656420426974636f696e000000000000000000008152506040518060400160405280600581526020017f54574254430000000000000000000000000000000000000000000000000000008152508160039081620000b5919062000534565b508060049081620000c7919062000534565b505050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16036200013f5760006040517f1e4fbdf70000000000000000000000000000000000000000000000000000000081526004016200013691906200062c565b60405180910390fd5b62000150816200015860201b60201c565b505062000649565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905081600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000620002508262000223565b9050919050565b620002628162000243565b81146200026e57600080fd5b50565b600081519050620002828162000257565b92915050565b600060208284031215620002a157620002a06200021e565b5b6000620002b18482850162000271565b91505092915050565b600081519050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806200033c57607f821691505b602082108103620003525762000351620002f4565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b600060088302620003bc7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826200037d565b620003c886836200037d565b95508019841693508086168417925050509392505050565b6000819050919050565b6000819050919050565b6000620004156200040f6200040984620003e0565b620003ea565b620003e0565b9050919050565b6000819050919050565b6200043183620003f4565b6200044962000440826200041c565b8484546200038a565b825550505050565b600090565b6200046062000451565b6200046d81848462000426565b505050565b5b8181101562000495576200048960008262000456565b60018101905062000473565b5050565b601f821115620004e457620004ae8162000358565b620004b9846200036d565b81016020851015620004c9578190505b620004e1620004d8856200036d565b83018262000472565b50505b505050565b600082821c905092915050565b60006200050960001984600802620004e9565b1980831691505092915050565b6000620005248383620004f6565b9150826002028217905092915050565b6200053f82620002ba565b67ffffffffffffffff8111156200055b576200055a620002c5565b5b62000567825462000323565b6200057482828562000499565b600060209050601f831160018114620005ac576000841562000597578287015190505b620005a3858262000516565b86555062000613565b601f198416620005bc8662000358565b60005b82811015620005e657848901518255600182019150602085019450602081019050620005bf565b8683101562000606578489015162000602601f891682620004f6565b8355505b6001600288020188555050505b505050505050565b620006268162000243565b82525050565b60006020820190506200064360008301846200061b565b92915050565b61136980620006596000396000f3fe608060405234801561001057600080fd5b50600436106101005760003560e01c806370a082311161009757806395d89b411161006657806395d89b4114610289578063a9059cbb146102a7578063dd62ed3e146102d7578063f2fde38b1461030757610100565b806370a0823114610215578063715018a61461024557806379cc67901461024f5780638da5cb5b1461026b57610100565b8063313ce567116100d3578063313ce567146101a157806340c10f19146101bf57806342966c68146101db5780635e8f1d94146101f757610100565b806306fdde0314610105578063095ea7b31461012357806318160ddd1461015357806323b872dd14610171575b600080fd5b61010d610323565b60405161011a9190610f90565b60405180910390f35b61013d6004803603810190610138919061104b565b6103b5565b60405161014a91906110a6565b60405180910390f35b61015b6103d8565b60405161016891906110d0565b60405180910390f35b61018b600480360381019061018691906110eb565b6103e2565b60405161019891906110a6565b60405180910390f35b6101a9610411565b6040516101b6919061115a565b60405180910390f35b6101d960048036038101906101d4919061104b565b61041a565b005b6101f560048036038101906101f09190611175565b61048a565b005b6101ff61049e565b60405161020c91906110d0565b60405180910390f35b61022f600480360381019061022a91906111a2565b6104a9565b60405161023c91906110d0565b60405180910390f35b61024d6104f1565b005b6102696004803603810190610264919061104b565b610505565b005b610273610525565b60405161028091906111de565b60405180910390f35b61029161054f565b60405161029e9190610f90565b60405180910390f35b6102c160048036038101906102bc919061104b565b6105e1565b6040516102ce91906110a6565b60405180910390f35b6102f160048036038101906102ec91906111f9565b610604565b6040516102fe91906110d0565b60405180910390f35b610321600480360381019061031c91906111a2565b61068b565b005b60606003805461033290611268565b80601f016020809104026020016040519081016040528092919081815260200182805461035e90611268565b80156103ab5780601f10610380576101008083540402835291602001916103ab565b820191906000526020600020905b81548152906001019060200180831161038e57829003601f168201915b5050505050905090565b6000806103c0610711565b90506103cd818585610719565b600191505092915050565b6000600254905090565b6000806103ed610711565b90506103fa85828561072b565b6104058585856107bf565b60019150509392505050565b60006008905090565b6104226108b3565b61042c828261093a565b660775f05a07400061043c6103d8565b11156104865761044a6103d8565b6040517ff602b06a00000000000000000000000000000000000000000000000000000000815260040161047d91906110d0565b60405180910390fd5b5050565b61049b610495610711565b826109bc565b50565b660775f05a07400081565b60008060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020549050919050565b6104f96108b3565b6105036000610a3e565b565b61051782610511610711565b8361072b565b61052182826109bc565b5050565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b60606004805461055e90611268565b80601f016020809104026020016040519081016040528092919081815260200182805461058a90611268565b80156105d75780601f106105ac576101008083540402835291602001916105d7565b820191906000526020600020905b8154815290600101906020018083116105ba57829003601f168201915b5050505050905090565b6000806105ec610711565b90506105f98185856107bf565b600191505092915050565b6000600160008473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905092915050565b6106936108b3565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16036107055760006040517f1e4fbdf70000000000000000000000000000000000000000000000000000000081526004016106fc91906111de565b60405180910390fd5b61070e81610a3e565b50565b600033905090565b6107268383836001610b04565b505050565b60006107378484610604565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81146107b957818110156107a9578281836040517ffb8f41b20000000000000000000000000000000000000000000000000000000081526004016107a093929190611299565b60405180910390fd5b6107b884848484036000610b04565b5b50505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036108315760006040517f96c6fd1e00000000000000000000000000000000000000000000000000000000815260040161082891906111de565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036108a35760006040517fec442f0500000000000000000000000000000000000000000000000000000000815260040161089a91906111de565b60405180910390fd5b6108ae838383610cdb565b505050565b6108bb610711565b73ffffffffffffffffffffffffffffffffffffffff166108d9610525565b73ffffffffffffffffffffffffffffffffffffffff1614610938576108fc610711565b6040517f118cdaa700000000000000000000000000000000000000000000000000000000815260040161092f91906111de565b60405180910390fd5b565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036109ac5760006040517fec442f050000000000000000000000000000000000000000000000000000000081526004016109a391906111de565b60405180910390fd5b6109b860008383610cdb565b5050565b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610a2e5760006040517f96c6fd1e000000000000000000000000000000000000000000000000000000008152600401610a2591906111de565b60405180910390fd5b610a3a82600083610cdb565b5050565b6000600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905081600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600073ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610b765760006040517fe602df05000000000000000000000000000000000000000000000000000000008152600401610b6d91906111de565b60405180910390fd5b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610be85760006040517f94280d62000000000000000000000000000000000000000000000000000000008152600401610bdf91906111de565b60405180910390fd5b81600160008673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020819055508015610cd5578273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92584604051610ccc91906110d0565b60405180910390a35b50505050565b600073ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610d2d578060026000828254610d2191906112ff565b92505081905550610e00565b60008060008573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002054905081811015610db9578381836040517fe450d38c000000000000000000000000000000000000000000000000000000008152600401610db093929190611299565b60405180910390fd5b8181036000808673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200190815260200160002081905550505b600073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610e495780600260008282540392505081905550610e96565b806000808473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600082825401925050819055505b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051610ef391906110d0565b60405180910390a3505050565b600081519050919050565b600082825260208201905092915050565b60005b83811015610f3a578082015181840152602081019050610f1f565b60008484015250505050565b6000601f19601f8301169050919050565b6000610f6282610f00565b610f6c8185610f0b565b9350610f7c818560208601610f1c565b610f8581610f46565b840191505092915050565b60006020820190508181036000830152610faa8184610f57565b905092915050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000610fe282610fb7565b9050919050565b610ff281610fd7565b8114610ffd57600080fd5b50565b60008135905061100f81610fe9565b92915050565b6000819050919050565b61102881611015565b811461103357600080fd5b50565b6000813590506110458161101f565b92915050565b6000806040838503121561106257611061610fb2565b5b600061107085828601611000565b925050602061108185828601611036565b9150509250929050565b60008115159050919050565b6110a08161108b565b82525050565b60006020820190506110bb6000830184611097565b92915050565b6110ca81611015565b82525050565b60006020820190506110e560008301846110c1565b92915050565b60008060006060848603121561110457611103610fb2565b5b600061111286828701611000565b935050602061112386828701611000565b925050604061113486828701611036565b9150509250925092565b600060ff82169050919050565b6111548161113e565b82525050565b600060208201905061116f600083018461114b565b92915050565b60006020828403121561118b5761118a610fb2565b5b600061119984828501611036565b91505092915050565b6000602082840312156111b8576111b7610fb2565b5b60006111c684828501611000565b91505092915050565b6111d881610fd7565b82525050565b60006020820190506111f360008301846111cf565b92915050565b600080604083850312156112105761120f610fb2565b5b600061121e85828601611000565b925050602061122f85828601611000565b9150509250929050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000600282049050600182168061128057607f821691505b60208210810361129357611292611239565b5b50919050565b60006060820190506112ae60008301866111cf565b6112bb60208301856110c1565b6112c860408301846110c1565b949350505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061130a82611015565b915061131583611015565b925082820190508082111561132d5761132c6112d0565b5b9291505056fea2646970667358221220504476fede93226437d3b11bb58ead86358659b4af71b44c02e326e137d37e1064736f6c63430008180033",
}

// TWBTCABI is the input ABI used to generate the binding from.
// Deprecated: Use TWBTCMetaData.ABI instead.
var TWBTCABI = TWBTCMetaData.ABI

// TWBTCBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TWBTCMetaData.Bin instead.
var TWBTCBin = TWBTCMetaData.Bin

// DeployTWBTC deploys a new Ethereum contract, binding an instance of TWBTC to it.
func DeployTWBTC(auth *bind.TransactOpts, backend bind.ContractBackend, owner_ common.Address) (common.Address, *types.Transaction, *TWBTC, error) {
	parsed, err := TWBTCMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TWBTCBin), backend, owner_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TWBTC{TWBTCCaller: TWBTCCaller{contract: contract}, TWBTCTransactor: TWBTCTransactor{contract: contract}, TWBTCFilterer: TWBTCFilterer{contract: contract}}, nil
}

// TWBTC is an auto generated Go binding around an Ethereum contract.
type TWBTC struct {
	TWBTCCaller     // Read-only binding to the contract
	TWBTCTransactor // Write-only binding to the contract
	TWBTCFilterer   // Log filterer for contract events
}

// TWBTCCaller is an auto generated read-only Go binding around an Ethereum contract.
type TWBTCCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TWBTCTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TWBTCTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TWBTCFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TWBTCFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TWBTCSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TWBTCSession struct {
	Contract     *TWBTC            // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TWBTCCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TWBTCCallerSession struct {
	Contract *TWBTCCaller  // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// TWBTCTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TWBTCTransactorSession struct {
	Contract     *TWBTCTransactor  // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TWBTCRaw is an auto generated low-level Go binding around an Ethereum contract.
type TWBTCRaw struct {
	Contract *TWBTC // Generic contract binding to access the raw methods on
}

// TWBTCCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TWBTCCallerRaw struct {
	Contract *TWBTCCaller // Generic read-only contract binding to access the raw methods on
}

// TWBTCTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TWBTCTransactorRaw struct {
	Contract *TWBTCTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTWBTC creates a new instance of TWBTC, bound to a specific deployed contract.
func NewTWBTC(address common.Address, backend bind.ContractBackend) (*TWBTC, error) {
	contract, err := bindTWBTC(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TWBTC{TWBTCCaller: TWBTCCaller{contract: contract}, TWBTCTransactor: TWBTCTransactor{contract: contract}, TWBTCFilterer: TWBTCFilterer{contract: contract}}, nil
}

// NewTWBTCCaller creates a new read-only instance of TWBTC, bound to a specific deployed contract.
func NewTWBTCCaller(address common.Address, caller bind.ContractCaller) (*TWBTCCaller, error) {
	contract, err := bindTWBTC(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TWBTCCaller{contract: contract}, nil
}

// NewTWBTCTransactor creates a new write-only instance of TWBTC, bound to a specific deployed contract.
func NewTWBTCTransactor(address common.Address, transactor bind.ContractTransactor) (*TWBTCTransactor, error) {
	contract, err := bindTWBTC(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TWBTCTransactor{contract: contract}, nil
}

// NewTWBTCFilterer creates a new log filterer instance of TWBTC, bound to a specific deployed contract.
func NewTWBTCFilterer(address common.Address, filterer bind.ContractFilterer) (*TWBTCFilterer, error) {
	contract, err := bindTWBTC(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TWBTCFilterer{contract: contract}, nil
}

// bindTWBTC binds a generic wrapper to an already deployed contract.
func bindTWBTC(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TWBTCMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TWBTC *TWBTCRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TWBTC.Contract.TWBTCCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TWBTC *TWBTCRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TWBTC.Contract.TWBTCTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TWBTC *TWBTCRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TWBTC.Contract.TWBTCTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TWBTC *TWBTCCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TWBTC.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TWBTC *TWBTCTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TWBTC.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TWBTC *TWBTCTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TWBTC.Contract.contract.Transact(opts, method, params...)
}

// MAXBTCSUPPLY is a free data retrieval call binding the contract method 0x5e8f1d94.
//
// Solidity: function MAX_BTC_SUPPLY() view returns(uint256)
func (_TWBTC *TWBTCCaller) MAXBTCSUPPLY(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "MAX_BTC_SUPPLY")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MAXBTCSUPPLY is a free data retrieval call binding the contract method 0x5e8f1d94.
//
// Solidity: function MAX_BTC_SUPPLY() view returns(uint256)
func (_TWBTC *TWBTCSession) MAXBTCSUPPLY() (*big.Int, error) {
	return _TWBTC.Contract.MAXBTCSUPPLY(&_TWBTC.CallOpts)
}

// MAXBTCSUPPLY is a free data retrieval call binding the contract method 0x5e8f1d94.
//
// Solidity: function MAX_BTC_SUPPLY() view returns(uint256)
func (_TWBTC *TWBTCCallerSession) MAXBTCSUPPLY() (*big.Int, error) {
	return _TWBTC.Contract.MAXBTCSUPPLY(&_TWBTC.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TWBTC *TWBTCCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TWBTC *TWBTCSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TWBTC.Contract.Allowance(&_TWBTC.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_TWBTC *TWBTCCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _TWBTC.Contract.Allowance(&_TWBTC.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TWBTC *TWBTCCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TWBTC *TWBTCSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _TWBTC.Contract.BalanceOf(&_TWBTC.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_TWBTC *TWBTCCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _TWBTC.Contract.BalanceOf(&_TWBTC.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint8)
func (_TWBTC *TWBTCCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint8)
func (_TWBTC *TWBTCSession) Decimals() (uint8, error) {
	return _TWBTC.Contract.Decimals(&_TWBTC.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() pure returns(uint8)
func (_TWBTC *TWBTCCallerSession) Decimals() (uint8, error) {
	return _TWBTC.Contract.Decimals(&_TWBTC.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TWBTC *TWBTCCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TWBTC *TWBTCSession) Name() (string, error) {
	return _TWBTC.Contract.Name(&_TWBTC.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_TWBTC *TWBTCCallerSession) Name() (string, error) {
	return _TWBTC.Contract.Name(&_TWBTC.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TWBTC *TWBTCCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TWBTC *TWBTCSession) Owner() (common.Address, error) {
	return _TWBTC.Contract.Owner(&_TWBTC.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_TWBTC *TWBTCCallerSession) Owner() (common.Address, error) {
	return _TWBTC.Contract.Owner(&_TWBTC.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TWBTC *TWBTCCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TWBTC *TWBTCSession) Symbol() (string, error) {
	return _TWBTC.Contract.Symbol(&_TWBTC.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_TWBTC *TWBTCCallerSession) Symbol() (string, error) {
	return _TWBTC.Contract.Symbol(&_TWBTC.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TWBTC *TWBTCCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TWBTC.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TWBTC *TWBTCSession) TotalSupply() (*big.Int, error) {
	return _TWBTC.Contract.TotalSupply(&_TWBTC.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_TWBTC *TWBTCCallerSession) TotalSupply() (*big.Int, error) {
	return _TWBTC.Contract.TotalSupply(&_TWBTC.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_TWBTC *TWBTCSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Approve(&_TWBTC.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Approve(&_TWBTC.TransactOpts, spender, value)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 value) returns()
func (_TWBTC *TWBTCTransactor) Burn(opts *bind.TransactOpts, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "burn", value)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 value) returns()
func (_TWBTC *TWBTCSession) Burn(value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Burn(&_TWBTC.TransactOpts, value)
}

// Burn is a paid mutator transaction binding the contract method 0x42966c68.
//
// Solidity: function burn(uint256 value) returns()
func (_TWBTC *TWBTCTransactorSession) Burn(value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Burn(&_TWBTC.TransactOpts, value)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 value) returns()
func (_TWBTC *TWBTCTransactor) BurnFrom(opts *bind.TransactOpts, account common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "burnFrom", account, value)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 value) returns()
func (_TWBTC *TWBTCSession) BurnFrom(account common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.BurnFrom(&_TWBTC.TransactOpts, account, value)
}

// BurnFrom is a paid mutator transaction binding the contract method 0x79cc6790.
//
// Solidity: function burnFrom(address account, uint256 value) returns()
func (_TWBTC *TWBTCTransactorSession) BurnFrom(account common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.BurnFrom(&_TWBTC.TransactOpts, account, value)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_TWBTC *TWBTCTransactor) Mint(opts *bind.TransactOpts, to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "mint", to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_TWBTC *TWBTCSession) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Mint(&_TWBTC.TransactOpts, to, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address to, uint256 amount) returns()
func (_TWBTC *TWBTCTransactorSession) Mint(to common.Address, amount *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Mint(&_TWBTC.TransactOpts, to, amount)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TWBTC *TWBTCTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TWBTC *TWBTCSession) RenounceOwnership() (*types.Transaction, error) {
	return _TWBTC.Contract.RenounceOwnership(&_TWBTC.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_TWBTC *TWBTCTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _TWBTC.Contract.RenounceOwnership(&_TWBTC.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Transfer(&_TWBTC.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.Transfer(&_TWBTC.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.TransferFrom(&_TWBTC.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_TWBTC *TWBTCTransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _TWBTC.Contract.TransferFrom(&_TWBTC.TransactOpts, from, to, value)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TWBTC *TWBTCTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _TWBTC.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TWBTC *TWBTCSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TWBTC.Contract.TransferOwnership(&_TWBTC.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_TWBTC *TWBTCTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _TWBTC.Contract.TransferOwnership(&_TWBTC.TransactOpts, newOwner)
}

// TWBTCApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the TWBTC contract.
type TWBTCApprovalIterator struct {
	Event *TWBTCApproval // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TWBTCApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TWBTCApproval)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TWBTCApproval)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TWBTCApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TWBTCApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TWBTCApproval represents a Approval event raised by the TWBTC contract.
type TWBTCApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TWBTC *TWBTCFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*TWBTCApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TWBTC.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &TWBTCApprovalIterator{contract: _TWBTC.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TWBTC *TWBTCFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *TWBTCApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _TWBTC.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TWBTCApproval)
				if err := _TWBTC.contract.UnpackLog(event, "Approval", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_TWBTC *TWBTCFilterer) ParseApproval(log types.Log) (*TWBTCApproval, error) {
	event := new(TWBTCApproval)
	if err := _TWBTC.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TWBTCOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the TWBTC contract.
type TWBTCOwnershipTransferredIterator struct {
	Event *TWBTCOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TWBTCOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TWBTCOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TWBTCOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TWBTCOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TWBTCOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TWBTCOwnershipTransferred represents a OwnershipTransferred event raised by the TWBTC contract.
type TWBTCOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TWBTC *TWBTCFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*TWBTCOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TWBTC.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &TWBTCOwnershipTransferredIterator{contract: _TWBTC.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TWBTC *TWBTCFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *TWBTCOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _TWBTC.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TWBTCOwnershipTransferred)
				if err := _TWBTC.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_TWBTC *TWBTCFilterer) ParseOwnershipTransferred(log types.Log) (*TWBTCOwnershipTransferred, error) {
	event := new(TWBTCOwnershipTransferred)
	if err := _TWBTC.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// TWBTCTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the TWBTC contract.
type TWBTCTransferIterator struct {
	Event *TWBTCTransfer // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TWBTCTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TWBTCTransfer)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TWBTCTransfer)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TWBTCTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TWBTCTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TWBTCTransfer represents a Transfer event raised by the TWBTC contract.
type TWBTCTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TWBTC *TWBTCFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*TWBTCTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TWBTC.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &TWBTCTransferIterator{contract: _TWBTC.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TWBTC *TWBTCFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *TWBTCTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _TWBTC.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TWBTCTransfer)
				if err := _TWBTC.contract.UnpackLog(event, "Transfer", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_TWBTC *TWBTCFilterer) ParseTransfer(log types.Log) (*TWBTCTransfer, error) {
	event := new(TWBTCTransfer)
	if err := _TWBTC.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
