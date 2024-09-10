package etherman

import (
	"context"
	"math/big"
	"strings"

	"github.com/0xPolygonHermez/zkevm-node/log"
	bridge "github.com/TEENet-io/bridge-go/contracts/TEENetBtcBridge"
	"github.com/TEENet-io/bridge-go/contracts/TWBTC"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
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

type Client struct {
	ethClient      ethereumClient
	bridgeAddress  common.Address
	bridgeContract *bridge.TEENetBtcBridge
	twbtcContract  *TWBTC.TWBTC
}

func NewClient(cfg *Config) (*Client, error) {
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		log.Errorf("error connecting to %s: %+v", cfg.URL, err)
		return nil, err
	}

	return &Client{
		ethClient:      ethClient,
		bridgeAddress:  common.HexToAddress(cfg.BridgeContractAddress),
		bridgeContract: nil,
		twbtcContract:  nil,
	}, nil
}

func (etherman *Client) GetLatestFinalizedBlockNumber() (*big.Int, error) {
	blk, err := etherman.ethClient.BlockByNumber(context.Background(), big.NewInt(-3))
	if err != nil {
		log.Errorf("error getting latest finalized block: %+v", err)
		return &big.Int{}, nil
	}
	return blk.Number(), nil
}

func (etherman *Client) GetEventMinted(blockNum *big.Int) (
	[]bridge.TEENetBtcBridgeMinted,
	[]bridge.TEENetBtcBridgeRedeemRequested,
	[]bridge.TEENetBtcBridgeRedeemPrepared,
	error,
) {
	logs, err := etherman.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: blockNum,
		ToBlock:   blockNum,
		Addresses: []common.Address{etherman.bridgeAddress},
	})
	if err != nil {
		log.Errorf("error getting logs: %+v", err)
		return nil, nil, nil, err
	}

	if len(logs) == 0 {
		return nil, nil, nil, nil
	}

	bridgeABI, err := abi.JSON(strings.NewReader(bridge.TEENetBtcBridgeABI))
	if err != nil {
		log.Errorf("error parsing ABI: %+v", err)
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
				log.Errorf("error unpacking Minted event: %+v", err)
				return nil, nil, nil, err
			}
			copy(ev.BtcTxId[:], vlog.Topics[1].Bytes())
			minted = append(minted, *ev)
		case RedeemRequestedSignatureHash:
			ev := new(bridge.TEENetBtcBridgeRedeemRequested)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemRequested", vlog.Data)
			if err != nil {
				log.Errorf("error unpacking RedeemRequested event: %+v", err)
				return nil, nil, nil, err
			}
			redeemRequested = append(redeemRequested, *ev)
		case RedeemPreparedSignatureHash:
			ev := new(bridge.TEENetBtcBridgeRedeemPrepared)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemPrepared", vlog.Data)
			if err != nil {
				log.Errorf("error unpacking RedeemPrepared event: %+v", err)
				return nil, nil, nil, err
			}
			copy(ev.EthTxHash[:], vlog.Topics[1].Bytes())
			redeemPrepared = append(redeemPrepared, *ev)
		default:
			log.Errorf("unknown event: %+v", vlog.Topics[0])
		}
	}

	return minted, redeemRequested, redeemPrepared, nil
}

func (etherman *Client) Mint(params *MintParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		log.Errorf("failed to get bridge contract: %+v", err)
		return err
	}

	btcTxId := HexStrToBytes32(string(params.BtcTxId))
	receiver := common.HexToAddress(string(params.Receiver))
	rx := HexStrToBigInt(string(params.Rx))
	s := HexStrToBigInt(string(params.S))

	_, err = contract.Mint(params.Auth, btcTxId, receiver, big.NewInt(int64(params.Amount)), rx, s)
	if err != nil {
		log.Errorf("failed to mint: %+v", err)
		return err
	}

	return nil
}

func (etherman *Client) RedeemRequest(params *RequestParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		log.Errorf("failed to get bridge contract: %+v", err)
		return err
	}

	receiver := string(params.Receiver)

	_, err = contract.RedeemRequest(params.Auth, big.NewInt(int64(params.Amount)), receiver)
	if err != nil {
		log.Errorf("failed to redeem requested: %+v", err)
		return err
	}

	return nil
}

func (etherman *Client) RedeemPrepare(params *PrepareParams) error {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		log.Errorf("failed to get bridge contract: %+v", err)
		return err
	}

	var outpointTxIds [][32]byte
	for _, txId := range params.OutpointTxIds {
		outpointTxIds = append(outpointTxIds, HexStrToBytes32(string(txId)))
	}

	redeemRequestTxHash := HexStrToBytes32(string(params.TxHash))
	requester := common.HexToAddress(string(params.Requester))
	amount := big.NewInt(int64(params.Amount))
	rx := HexStrToBigInt(params.Rx)
	s := HexStrToBigInt(params.S)

	_, err = contract.RedeemPrepare(
		params.Auth,
		redeemRequestTxHash,
		requester,
		amount,
		outpointTxIds,
		params.OutpointIdxs,
		rx,
		s,
	)
	if err != nil {
		log.Errorf("failed to redeem prepared: %+v", err)
		return err
	}

	return nil
}

func (etherman *Client) TWBTCAddress() (common.Address, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		log.Errorf("failed to get bridge contract: %+v", err)
		return common.Address{}, err
	}

	twbtcAddr, err := contract.Twbtc(nil)
	if err != nil {
		log.Errorf("failed to get TWBTC address: %+v", err)
		return common.Address{}, err
	}

	return twbtcAddr, nil
}

func (etherman *Client) TWBTCBalanceOf(addr common.Address) (*big.Int, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		log.Errorf("failed to get TWBTC contract: %+v", err)
		return nil, err
	}

	balance, err := contract.BalanceOf(nil, addr)
	if err != nil {
		log.Errorf("failed to get TWBTC balance: %+v", err)
		return nil, err
	}

	return balance, nil
}

func (etherman *Client) TWBTCApprove(auth *bind.TransactOpts, amount *big.Int) error {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		log.Errorf("failed to get TWBTC contract: %+v", err)
		return err
	}

	_, err = contract.Approve(auth, etherman.bridgeAddress, amount)
	if err != nil {
		log.Errorf("failed to approve TWBTC: %+v", err)
		return err
	}

	return nil
}

func (etherman *Client) TWBTCAllowance(owner common.Address) (*big.Int, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		log.Errorf("failed to get TWBTC contract: %+v", err)
		return nil, err
	}

	allowance, err := contract.Allowance(nil, owner, etherman.bridgeAddress)
	if err != nil {
		log.Errorf("failed to get TWBTC allowance: %+v", err)
		return nil, err
	}

	return allowance, nil
}

func (etherman *Client) getBridgeContract() (*bridge.TEENetBtcBridge, error) {
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

func (etherman *Client) getTWBTCContract() (*TWBTC.TWBTC, error) {
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
