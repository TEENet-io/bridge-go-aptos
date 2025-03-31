package etherman

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

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
	logger "github.com/sirupsen/logrus"
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

	cfg *EthermanConfig

	auth *bind.TransactOpts // ethereum account controlled by bridge.
	mu   sync.Mutex
}

// TODO: Un-tested
// Create a new Etherman instance.
// auth is used to sign bridge txs (mint, redeemRequest, redeemPrepare).
// So auth should have some eth (as gas) within it.
func NewEtherman(cfg *EthermanConfig, auth *bind.TransactOpts) (*Etherman, error) {
	ethClient, err := ethclient.Dial(cfg.URL)
	if err != nil {
		logger.WithField("url", cfg.URL).Errorf("failed to dial Ethereum node: err=%v", err)
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

// !!! Bridge initiates this tx !!!
// Call the function mint() on smart contract.
// This really generates TWBTC to user on the chain.
func (etherman *Etherman) Mint(params *MintParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	// update nonce before sending tx
	etherman.mu.Lock()
	defer etherman.mu.Unlock()
	nonce, err := etherman.getAuthNonce()
	if err != nil {
		return nil, err
	}
	etherman.auth.Nonce = new(big.Int).SetUint64(nonce)

	return contract.Mint(
		etherman.auth,
		params.BtcTxId,
		ethcommon.BytesToAddress(params.Receiver),
		params.Amount,
		params.Rx,
		params.S,
	)
}

// !!! User initiates this tx !!!
// Call the function redeemRequest() on smart contract.
// sender = auth. (a user/client, ethereum side, the twbtc owner)
// amount = params.Amount
// receiver = params.Receiver (btc address)
func (etherman *Etherman) RedeemRequest(auth *bind.TransactOpts, params *RequestParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	return contract.RedeemRequest(auth, params.Amount, string(params.Receiver))
}

// Call "Redeem Prepare" on the bridge contract.
// This writes a set of BTC UTXOs = list[(txid, vout)] on the bridge contract.
func (etherman *Etherman) RedeemPrepare(params *PrepareParams) (*types.Transaction, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return nil, err
	}

	// convert ethcommon.Hash to [32]byte
	outpointTxIds := [][32]byte{} // a slice of [32]byte
	for _, txid := range params.OutpointTxIds {
		outpointTxIds = append(outpointTxIds, txid)
	}

	// udpate nonce before sending tx
	etherman.mu.Lock()
	defer etherman.mu.Unlock()
	nonce, err := etherman.getAuthNonce()
	if err != nil {
		return nil, err
	}
	etherman.auth.Nonce = new(big.Int).SetUint64(nonce)

	return contract.RedeemPrepare(
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

// Approve from auth (owner) the amount of TWBTC that can be spent by our bridge (spender).
func (etherman *Etherman) TWBTCApprove(auth *bind.TransactOpts, amount *big.Int) (*types.Transaction, error) {
	contract, err := etherman.getTWBTCContract()
	if err != nil {
		return nil, err
	}

	return contract.Approve(auth, etherman.cfg.BridgeContractAddress, amount)
}

// Check the allowance of TWBTC (from owner) that can be spent by our bridge (spender).
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

func (etherman *Etherman) IsUsed(btcTxId [32]byte) (bool, error) {
	contract, err := etherman.getBridgeContract()
	if err != nil {
		return false, err
	}

	ok, err := contract.IsUsed(nil, btcTxId)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (etherman *Etherman) getAuthNonce() (uint64, error) {
	nonce, err := etherman.ethClient.PendingNonceAt(context.Background(), etherman.auth.From)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (etherman *Etherman) OnCanonicalChain(hash ethcommon.Hash) (bool, error) {
	// get the block number of the given hash
	header, err := etherman.ethClient.HeaderByHash(context.Background(), hash)
	if err != nil {
		return false, err
	}
	blkNum := header.Number

	// get the canonical block header of the given block number
	canonicalHeader, err := etherman.ethClient.HeaderByNumber(context.Background(), blkNum)
	if err != nil {
		return false, err
	}

	// compare the given hash with the canonical block header hash
	return hash == canonicalHeader.Hash(), nil
}

// Fetch the newest eth balance of addr, in satoshi.
func (etherman *Etherman) GetBalance(addr ethcommon.Address) (*big.Int, error) {
	balance, err := etherman.ethClient.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// Wait for the tx to be mined and get the receipt.
// If the tx is reverted, return false+nil.
// If the tx is successful, return true+nil.
// This is a blocking function.
// If rpc call unsuccesful, return false + err
// If waiting timeout, will also return false + error
func (etherman *Etherman) WaitForTxReceipt(tx *types.Transaction, retryTimes int, retryInterval int) (bool, error) {
	RETRY_INTERVAL := time.Duration(retryInterval) * time.Second // seconds
	for i := 0; i < retryTimes; i++ {
		receipt, err := etherman.ethClient.TransactionReceipt(context.Background(), tx.Hash())

		if err != nil {
			// if not found receipt just poll again.
			if err.Error() == ethereum.NotFound.Error() {
				time.Sleep(RETRY_INTERVAL)
				continue
			} else { // other rpc error then report the error.
				return false, err
			}
		}
		if receipt != nil {
			if receipt.Status == 1 {
				logger.WithFields(logger.Fields{
					"evm_txid":   tx.Hash().Hex(),
					"block_hash": receipt.BlockHash.Hex(),
					"block_num":  receipt.BlockNumber.Int64(),
				}).Info("Transaction executed successfully")
				return true, nil
			} else {
				logger.WithFields(logger.Fields{
					"evm_txid":   tx.Hash().Hex(),
					"block_hash": receipt.BlockHash.Hex(),
					"block_num":  receipt.BlockNumber.Int64(),
				}).Error("Transaction reverted")
				return false, nil
			}
		}
	}
	return false, errors.New("timeout waiting for tx receipt: " + tx.Hash().Hex())
}
