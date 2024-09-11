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
	[]bridge.TEENetBtcBridgeMinted,
	[]bridge.TEENetBtcBridgeRedeemRequested,
	[]bridge.TEENetBtcBridgeRedeemPrepared,
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

	minted := make([]bridge.TEENetBtcBridgeMinted, 0, len(logs))
	redeemRequested := make([]bridge.TEENetBtcBridgeRedeemRequested, 0, len(logs))
	redeemPrepared := make([]bridge.TEENetBtcBridgeRedeemPrepared, 0, len(logs))

	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case MintedSignatureHash:
			ev := new(bridge.TEENetBtcBridgeMinted)
			err = bridgeABI.UnpackIntoInterface(ev, "Minted", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.BtcTxId[:], vlog.Topics[1].Bytes())
			minted = append(minted, *ev)
		case RedeemRequestedSignatureHash:
			ev := new(bridge.TEENetBtcBridgeRedeemRequested)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemRequested", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			redeemRequested = append(redeemRequested, *ev)
		case RedeemPreparedSignatureHash:
			ev := new(bridge.TEENetBtcBridgeRedeemPrepared)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemPrepared", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.EthTxHash[:], vlog.Topics[1].Bytes())
			redeemPrepared = append(redeemPrepared, *ev)
		default:
			return nil, nil, nil, fmt.Errorf("unknown event: %+v", vlog.Topics[0])
		}
	}

	return minted, redeemRequested, redeemPrepared, nil
}

func (etherman *Etherman) Mint(params *MintParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return err
	}

	btcTxId := HexStrToBytes32(string(params.BtcTxId))
	receiver := ethcommon.HexToAddress(string(params.Receiver))
	rx := HexStrToBigInt(string(params.Rx))
	s := HexStrToBigInt(string(params.S))

	_, err = contract.Mint(params.Auth, btcTxId, receiver, params.Amount, rx, s)
	if err != nil {
		return err
	}

	return nil
}

func (etherman *Etherman) RedeemRequest(params *RequestParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return err
	}

	receiver := string(params.Receiver)

	_, err = contract.RedeemRequest(params.Auth, params.Amount, receiver)
	if err != nil {
		return err
	}

	return nil
}

func (etherman *Etherman) RedeemPrepare(params *PrepareParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return err
	}

	var outpointTxIds [][32]byte
	for _, txId := range params.OutpointTxIds {
		outpointTxIds = append(outpointTxIds, HexStrToBytes32(string(txId)))
	}

	redeemRequestTxHash := HexStrToBytes32(string(params.TxHash))
	requester := ethcommon.HexToAddress(string(params.Requester))
	rx := HexStrToBigInt(params.Rx)
	s := HexStrToBigInt(params.S)

	_, err = contract.RedeemPrepare(
		params.Auth,
		redeemRequestTxHash,
		requester,
		params.Amount,
		outpointTxIds,
		params.OutpointIdxs,
		rx,
		s,
	)
	if err != nil {
		return err
	}

	return nil
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
