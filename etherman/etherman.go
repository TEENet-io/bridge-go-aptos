package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
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
	RedeemPreparedSignatureHash  = crypto.Keccak256Hash([]byte("RedeemPrepared(bytes32,address,string,uint256,bytes32[],uint16[])"))
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

	ChainID(context.Context) (*big.Int, error)
}

type Etherman struct {
	ethClient ethereumClient

	cfg *Config

	auth *bind.TransactOpts
	mu   sync.Mutex
}

func NewEtherman(cfg *Config, auth *bind.TransactOpts) (*Etherman, error) {
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		logger.Errorf("failed to dial Ethereum node: url=%s, err=%v", cfg.URL, err)
		return nil, err
	}

	contract, err := bridge.NewTEENetBtcBridge(cfg.BridgeContractAddress, ethClient)
	if err != nil {
		logger.Errorf("failed to create bridge contract instance: address=0x%x, err=%v", cfg.BridgeContractAddress, err)
		return nil, err
	}

	twbtcAddr, err := contract.Twbtc(nil)
	if err != nil {
		logger.Errorf("failed to get TWBTC address from bridge contract: err=%v", err)
		return nil, err
	}

	if twbtcAddr != cfg.TWBTCContractAddress {
		errMsg := fmt.Sprintf("TWBTC address mismatch: expected=0x%x, got=0x%x", cfg.TWBTCContractAddress, twbtcAddr)
		logger.Errorf(errMsg)
		return nil, errors.New(errMsg)
	}

	return &Etherman{
		ethClient: ethClient,
		cfg:       cfg,
		auth:      auth,
	}, nil
}

func (etherman *Etherman) Client() ethereumClient {
	return etherman.ethClient
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
	[]MintedEvent,
	[]RedeemRequestedEvent,
	[]RedeemPreparedEvent,
	error,
) {
	logs, err := etherman.ethClient.FilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: blockNum,
		ToBlock:   blockNum,
		Addresses: []ethcommon.Address{etherman.cfg.BridgeContractAddress},
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

	minted := make([]MintedEvent, 0, len(logs))
	redeemRequested := make([]RedeemRequestedEvent, 0, len(logs))
	redeemPrepared := make([]RedeemPreparedEvent, 0, len(logs))

	for _, vlog := range logs {
		switch vlog.Topics[0] {
		case MintedSignatureHash:
			ev := new(MintedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "Minted", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.BtcTxId[:], vlog.Topics[1].Bytes())
			copy(ev.TxHash[:], vlog.TxHash.Bytes())
			minted = append(minted, *ev)
		case RedeemRequestedSignatureHash:
			ev := new(RedeemRequestedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemRequested", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.TxHash[:], vlog.TxHash.Bytes())
			redeemRequested = append(redeemRequested, *ev)
		case RedeemPreparedSignatureHash:
			ev := new(RedeemPreparedEvent)
			err = bridgeABI.UnpackIntoInterface(ev, "RedeemPrepared", vlog.Data)
			if err != nil {
				return nil, nil, nil, err
			}
			copy(ev.EthTxHash[:], vlog.Topics[1].Bytes())
			copy(ev.TxHash[:], vlog.TxHash.Bytes())
			redeemPrepared = append(redeemPrepared, *ev)
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

	// Prevent auth from being modified by other goroutines
	etherman.mu.Lock()
	defer etherman.mu.Unlock()

	tx, err := contract.Mint(
		etherman.auth,
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

func (etherman *Etherman) RedeemRequest(auth *bind.TransactOpts, params *RequestParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	tx, err := contract.RedeemRequest(auth, params.Amount, string(params.Receiver))
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

	outpointTxIds := [][32]byte{}
	for _, txid := range params.OutpointTxIds {
		outpointTxIds = append(outpointTxIds, txid)
	}

	// Prevent auth from being modified by other goroutines
	etherman.mu.Lock()
	defer etherman.mu.Unlock()

	tx, err := contract.RedeemPrepare(
		etherman.auth,
		params.RequestTxHash,
		params.Requester,
		string(params.Receiver),
		params.Amount,
		outpointTxIds,
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

func (etherman *Etherman) TWBTCApprove(auth *bind.TransactOpts, amount *big.Int) (ethcommon.Hash, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return ethcommon.Hash{}, err
	}

	tx, err := contract.Approve(auth, etherman.cfg.BridgeContractAddress, amount)
	if err != nil {
		return ethcommon.Hash{}, err
	}

	return tx.Hash(), nil
}

func (etherman *Etherman) TWBTCAllowance(owner ethcommon.Address) (*big.Int, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return nil, err
	}

	allowance, err := contract.Allowance(nil, owner, etherman.cfg.BridgeContractAddress)
	if err != nil {
		return nil, err
	}

	return allowance, nil
}

func (etherman *Etherman) GetPublicKey() (*big.Int, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	pk, err := contract.Pk(nil)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

func (etherman *Etherman) getBridgeContract() (*bridge.TEENetBtcBridge, error) {
	contract, err := bridge.NewTEENetBtcBridge(etherman.cfg.BridgeContractAddress, etherman.ethClient)
	if err != nil {
		return nil, err
	}

	return contract, nil
}

func (etherman *Etherman) getTWBTCContract() (*TWBTC.TWBTC, error) {
	contract, err := TWBTC.NewTWBTC(etherman.cfg.TWBTCContractAddress, etherman.ethClient)
	if err != nil {
		return nil, err
	}

	return contract, nil
}

func (etherman *Etherman) IsPrepared(ethTxHash [32]byte) (bool, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return false, err
	}

	ok, err := contract.IsPrepared(nil, ethTxHash)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (etherman *Etherman) IsMinted(btcTxId [32]byte) (bool, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return false, err
	}

	ok, err := contract.IsMinted(nil, btcTxId)
	if err != nil {
		return false, err
	}

	return ok, nil
}
func (etherman *Etherman) SetNonce(nonce uint64) {
	etherman.mu.Lock()
	defer etherman.mu.Unlock()

	etherman.auth.Nonce = big.NewInt(int64(nonce))
}

func (etherman *Etherman) SetGasLimit(limit uint64) {
	etherman.mu.Lock()
	defer etherman.mu.Unlock()

	etherman.auth.GasLimit = limit
}

func (etherman *Etherman) SetGasPrice(price *big.Int) {
	etherman.mu.Lock()
	defer etherman.mu.Unlock()

	etherman.auth.GasPrice = common.BigIntClone(price)
}

func (etherman *Etherman) UpdateBackendAccountNonce() error {
	nonce, err := etherman.ethClient.PendingNonceAt(context.Background(), etherman.auth.From)
	if err != nil {
		return err
	}

	etherman.SetNonce(nonce)

	return nil
}
