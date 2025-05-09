package rpc

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
)

const (
	MAX_CONFIRM = 9999999
)

type RpcClientConfig struct {
	ServerAddr string // ip address of server
	Port       string // port of server
	Username   string
	Pwd        string
}

// Wrapper of btc rpc client.
type RpcClient struct {
	ServerAddr string // ip address of server
	Port       string // port of server
	Username   string
	Pwd        string
	client     *rpcclient.Client
}

// Create a new RPC client which
// contains several useful functions
// to interact with bitcoin node.
func NewRpcClient(rcc *RpcClientConfig) (*RpcClient, error) {
	// Connect to local Bitcoin mining node using HTTP
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         rcc.ServerAddr + ":" + rcc.Port,
		User:         rcc.Username,
		Pass:         rcc.Pwd,
		HTTPPostMode: true, // original bitcoin only supports HTTP POST mode
		DisableTLS:   true, // original bitcoin does not support TLS
	}, nil)

	if err != nil {
		return nil, err
	}

	return &RpcClient{rcc.ServerAddr, rcc.Port, rcc.Username, rcc.Pwd, client}, nil
}

// Close the rpc client
func (r *RpcClient) Close() {
	r.client.Shutdown()
}

// Fetch a raw tx with a given TxID.
// Enable -txindex on your bitcoin node before using this function.
func (r *RpcClient) GetTx(TxID string) (*btcutil.Tx, error) {
	txHash, err := chainhash.NewHashFromStr(TxID)
	if err != nil {
		return nil, err
	}
	txRaw, err := r.client.GetRawTransaction(txHash)
	if err != nil {
		return nil, err
	}
	return txRaw, nil
}

// Get the latest block height.
func (r *RpcClient) GetLatestBlockHeight() (int64, error) {
	latestHeight, err := r.client.GetBlockCount()
	if err != nil {
		return 0, err
	}
	return latestHeight, nil
}

// Get the block height by providing block hash.
func (r *RpcClient) GetBlockHeightByHash(blockHash *chainhash.Hash) (int32, error) {
	blockHeaderVerbose, err := r.client.GetBlockHeaderVerbose(blockHash)
	if err != nil {
		return 0, err
	}

	// Get the block height
	blockHeight := blockHeaderVerbose.Height
	return blockHeight, nil
}

func reverse[T any](s []T) {
	n := len(s)
	for i := 0; i < n/2; i++ {
		s[i], s[n-i-1] = s[n-i-1], s[i]
	}
}

// Fetch nearest n blocks that is finalized (at least offset blocks old).
// Specify the amount of blocks to retrieve via n.
// Specify the offset (maturity, suggest 6) via offset.
// Return blocks is ordered from new to old.
func (r *RpcClient) GetBlocks(n int, offset int) ([]*wire.MsgBlock, error) {
	// latest height of blockchain
	latestHeight, err := r.client.GetBlockCount()
	if err != nil {
		return nil, err
	}
	// check up
	if offset < 0 || n <= 0 {
		return nil, fmt.Errorf("invalid offset or number of blocks: offset=%d, n=%d", offset, n)
	}

	var myBlocks []*wire.MsgBlock

	for i := 0; i < n; i++ {
		targetHeight := int(latestHeight) - offset - i
		if targetHeight < 0 {
			return nil, fmt.Errorf("latest height is %d, requested %d blocks with offset %d", latestHeight, n, offset)
		}
		hash, err := r.client.GetBlockHash(int64(targetHeight))
		if err != nil {
			return nil, err
		}
		b, err := r.client.GetBlock(hash)
		if err != nil {
			return nil, err
		}
		myBlocks = append(myBlocks, b)
	}
	if len(myBlocks) > 1 {
		reverse(myBlocks)
	}
	return myBlocks, nil
}

// Get the UTXO(s) of an address.
// Notice: You need to turn on option -txindex on bitcoin node for node to track the UTXO(s).
// Notice: This is not very accurate, btc nodes tend to forget to track.
// Notice: This won't scale well once the query goes very large.
// Notice: You fill in either P2PKH or P2WPKH address, the result is specific to that address type.
func (r *RpcClient) GetUtxoList(myAddress btcutil.Address, offset int) ([]utxo.UTXO, error) {
	// Get the list of unspent transaction outputs
	unspentOutputs, err := r.client.ListUnspentMinMaxAddresses(offset, MAX_CONFIRM, []btcutil.Address{myAddress})
	if err != nil {
		return nil, err
	}

	var u []utxo.UTXO
	for _, item := range unspentOutputs {
		txRaw, err := r.GetTx(item.TxID)
		if err != nil {
			return nil, err
		}
		outputPoint := txRaw.MsgTx().TxOut[item.Vout]

		var pkType utxo.PubKeyScriptType
		if txscript.IsPayToPubKeyHash(outputPoint.PkScript) {
			pkType = utxo.P2PKH_SCRIPT_T
		} else if txscript.IsPayToWitnessPubKeyHash(outputPoint.PkScript) {
			pkType = utxo.P2WPKH_SCRIPT_T
		} else {
			pkType = utxo.ANY_SCRIPT_T
		}

		u = append(u, utxo.UTXO{TxID: item.TxID, TxHash: txRaw.Hash(), Vout: item.Vout, Amount: outputPoint.Value, PkScriptT: pkType, PkScript: outputPoint.PkScript})
	}
	return u, nil
}

// Send raw transaction to bitcoin network.
func (r *RpcClient) SendRawTx(tx *wire.MsgTx) (*chainhash.Hash, error) {
	// Explanation on allowHighFees=true
	// It is a protection.
	// if bitcoin node thinks your fee is too high (maybe due to program mistakes) it can reject you.
	// false = may reject; true = accept it anyway
	txHash, err := r.client.SendRawTransaction(tx, true)
	return txHash, err
}

// Unfortunately there is no direct "get balance of an address" on btc node.
// To get the total balance of an address,
// this function sums up the value of all UTXOs associated with the given address.
// Note: if balance = 0, it can mean
// 1) the address really doesn't have any money.
// 2) the address is not tracked by the node.
func (r *RpcClient) GetBalance(myAddress btcutil.Address, offset int) (int64, error) {
	utxos, err := r.GetUtxoList(myAddress, offset)
	if err != nil {
		return 0, err
	}

	var totalBalance int64
	for _, utxo := range utxos {
		totalBalance += utxo.Amount
	}

	return totalBalance, nil
}

// Import a private key to the Bitcoin node's wallet.
// This is a deprecated feature in Bitcoin Node >=27 (since they move out of private key model to HD-Wallet Model)
// In Bitcoin Node <= 0.21, it is the normal behaviour.
// For btc-node > 27, node software should start btc node with -deprecatedrpc=create_bdb to enable this feature.
// And when you create a wallet for node, use `createwallet wallet1 false false "" false false`
// You can leave the lable empty string if just tracking.
// Note: If the priv key exists, it won't raise exception.
func (r *RpcClient) ImportPrivateKey(wif *btcutil.WIF, label string) error {
	err := r.client.ImportPrivKeyRescan(wif, label, true)
	if err != nil {
		return err
	}
	return nil
}

// Tell the <btc node> to track txs an address.
// This is a deprecated feature in Bitcoin Node >=27 (since they move out of private key model to HD-Wallet Model)
// In Bitcoin Node <= 0.21, it is the normal behaviour.
// For btc-node > 27, node software should start btc node with -deprecatedrpc=create_bdb to enable this feature.
// And when you create a wallet for node, use `createwallet wallet1 false false "" false false`
// In btc-node >= 27, Use import descriptor instead.
// See: https://github.com/bitcoin/bitcoin/issues/29772
// You can leave the lable empty string if just tracking.
func (r *RpcClient) ImportAddress(address string, label string) error {
	err := r.client.ImportAddressRescan(address, label, true)
	if err != nil {
		return err
	}
	return nil
}

// Generate a given number of blocks.
// This function is useful in testing environment.
// Return value is a list of block hashes generated.
func (r *RpcClient) GenerateBlocks(numBlocks int64, coinbase btcutil.Address) ([]*chainhash.Hash, error) {
	// Note: the original r.client.Generate() is deprecated in the library.
	blockHashes, err := r.client.GenerateToAddress(numBlocks, coinbase, nil)
	if err != nil {
		return nil, err
	}
	return blockHashes, nil
}
