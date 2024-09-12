package etherman

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/TEENet-io/bridge-go/common"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/TEENet-io/bridge-go/contracts/TWBTC"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// Events
	MintedSignatureHash          = crypto.Keccak256Hash([]byte("Minted(bytes32,address,uint256)"))
	RedeemRequestedSignatureHash = crypto.Keccak256Hash([]byte("RedeemRequested(address,uint256,string)"))
	RedeemPreparedSignatureHash  = crypto.Keccak256Hash([]byte("RedeemPrepared(bytes32,address,uint256,bytes32[],uint16[])"))
)

type ethereumClient interface {
	ethereum.ChainReader
	ethereum.ChainStateReader
	ethereum.ContractCaller
	ethereum.GasEstimator
	ethereum.GasPricer
	ethereum.LogFilterer
	ethereum.TransactionReader
	ethereum.TransactionSender

	bind.DeployBackend
	bind.ContractBackend
}

type Etherman struct {
	ethClient      ethereumClient
	bridgeAddress  ethcommon.Address
	bridgeContract *bridge.TEENetBtcBridge
	twbtcContract  *TWBTC.TWBTC
}

func NewEtherman(cfg *Config) (*Etherman, error) {
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}

	return &Etherman{
		ethClient:      ethClient,
		bridgeAddress:  ethcommon.HexToAddress(cfg.BridgeContractAddress),
		bridgeContract: nil,
		twbtcContract:  nil,
	}, nil
}

func (etherman *Etherman) GetLatestFinalizedBlockNumber() (*big.Int, error) {
	if common.Debug {
		blk, err := etherman.ethClient.BlockByNumber(context.Background(), nil)
		if err != nil {
			return nil, err
		}
		return blk.Number(), nil
	}

	blk, err := etherman.ethClient.BlockByNumber(context.Background(), big.NewInt(-3))
	if err != nil {
		return nil, err
	}
	return blk.Number(), nil
}

func (etherman *Etherman) GetEventLogs(blockNum *big.Int) (
	map[string]*MintedEvent,
	map[string]*RedeemRequestedEvent,
	map[string]*RedeemPreparedEvent,
	error,
) {
	logs, err := etherman.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: blockNum,
		ToBlock:   blockNum,
		Addresses: []ethcommon.Address{etherman.bridgeAddress},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if len(logs) == 0 {
		return nil, nil, nil, nil
	}

	bridgeABI, err := abi.JSON(strings.NewReader(bridge.TEENetBtcBridgeABI))
	if err != nil {
		return nil, nil, nil, err
	}

	minted := make(map[string]*MintedEvent)
	redeemRequested := make(map[string]*RedeemRequestedEvent)
	redeemPrepared := make(map[string]*RedeemPreparedEvent)

	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case MintedSignatureHash:
			ev := new(MintedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "Minted", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.BtcTxId[:], vlog.Topics[1].Bytes())
			minted[ethcommon.Bytes2Hex(ev.BtcTxId[:])] = ev
		case RedeemRequestedSignatureHash:
			ev := new(RedeemRequestedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemRequested", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.TxHash[:], vlog.TxHash.Bytes())
			redeemRequested[ethcommon.Bytes2Hex(vlog.TxHash.Bytes())] = ev
		case RedeemPreparedSignatureHash:
			ev := new(RedeemPreparedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemPrepared", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.EthTxHash[:], vlog.Topics[1].Bytes())
			redeemPrepared[ethcommon.Bytes2Hex(ev.EthTxHash[:])] = ev
		default:
			return nil, nil, nil, fmt.Errorf("unknown event: %+v", vlog.Topics[0])
		}
	}

	return minted, redeemRequested, redeemPrepared, nil
}

func (etherman *Etherman) Mint(params *MintParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	tx, err := contract.Mint(
		params.Auth,
		params.BtcTxId,
		params.Receiver,
		params.Amount,
		params.Rx,
		params.S,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (etherman *Etherman) RedeemRequest(params *RequestParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	tx, err := contract.RedeemRequest(params.Auth, params.Amount, string(params.Receiver))
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (etherman *Etherman) RedeemPrepare(params *PrepareParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	tx, err := contract.RedeemPrepare(
		params.Auth,
		params.TxHash,
		params.Requester,
		params.Amount,
		params.OutpointTxIds,
		params.OutpointIdxs,
		params.Rx,
		params.S,
	)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (etherman *Etherman) TWBTCAddress() (ethcommon.Address, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return ethcommon.Address{}, err
	}

	twbtcAddr, err := contract.Twbtc(nil)
	if err != nil {
		return ethcommon.Address{}, err
	}

	return twbtcAddr, nil
}

func (etherman *Etherman) TWBTCBalanceOf(addr ethcommon.Address) (*big.Int, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return nil, err
	}

	balance, err := contract.BalanceOf(nil, addr)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (etherman *Etherman) TWBTCApprove(auth *bind.TransactOpts, amount *big.Int) error {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return err
	}

	_, err = contract.Approve(auth, etherman.bridgeAddress, amount)
	if err != nil {
		return err
	}

	return nil
}

func (etherman *Etherman) TWBTCAllowance(owner ethcommon.Address) (*big.Int, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return nil, err
	}

	allowance, err := contract.Allowance(nil, owner, etherman.bridgeAddress)
	if err != nil {
		return nil, err
	}

	return allowance, nil
}

func (etherman *Etherman) getBridgeContract() (*bridge.TEENetBtcBridge, error) {
	if etherman.bridgeContract != nil {
		return etherman.bridgeContract, nil
	}

	contract, err := bridge.NewTEENetBtcBridge(etherman.bridgeAddress, etherman.ethClient)
	if err != nil {
		return nil, err
	}

	etherman.bridgeContract = contract

	return contract, nil
}

func (etherman *Etherman) getTWBTCContract() (*TWBTC.TWBTC, error) {
	if etherman.twbtcContract != nil {
		return etherman.twbtcContract, nil
	}

	twbtcAddr, err := etherman.TWBTCAddress()
	if err != nil {
		return nil, err
	}

	etherman.twbtcContract, err = TWBTC.NewTWBTC(twbtcAddr, etherman.ethClient)
	if err != nil {
		return nil, err
	}

	return etherman.twbtcContract, nil
}
