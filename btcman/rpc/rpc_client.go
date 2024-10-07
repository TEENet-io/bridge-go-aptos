package btcman

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
)

const (
	CONFIRM_THRESHOLD = 6 // minimum confirm threshold to consider Tx is finalized.
)

type RpcClient struct {
	ServerAddr string // ip address of server
	Port       string // port of server
	User       string
	Pass       string
	client     *rpcclient.Client
}

// Create a new RPC client which
// contains several useful functions
// to interact with bitcoin node.
func NewRpcClient(server string, port string, username string, password string) (*RpcClient, error) {
	// Connect to local Bitcoin mining node using HTTP
	client, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         server + ":" + port,
		User:         username,
		Pass:         password,
		HTTPPostMode: true,
		DisableTLS:   true,
	}, nil)

	if err != nil {
		return nil, err
	}

	return &RpcClient{server, port, username, password, client}, nil
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

// Fetch nearest n blocks that is surely finalized (at least offset blocks old).
// Specify the amount of blocks to retrieve via n.
// Specify the offset (maturity, suggest 6) via offset.
// Return blocks is ordered from new to old.
func (r *RpcClient) GetFinalizedBlocks(n int, offset int) ([]*wire.MsgBlock, error) {
	// latest height of blockchain
	latestHeight, err := r.client.GetBlockCount()
	if err != nil {
		return nil, err
	}
	// check up
	if offset < 0 || n <= 0 {
		return nil, fmt.Errorf("invalid offset or number of blocks: offset=%d, n=%d", offset, n)
	}
	// allocate slice of n
	myBlocks := make([]*wire.MsgBlock, n)

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
	return myBlocks, nil
}

// FindDepositTx searches for tx that output #1 = money to us, output #2 = OP_RETURN, output #3 = we don't care ...
func (r *RpcClient) FindDepositTx(block *wire.MsgBlock, targetAddress btcutil.Address, chainParams *chaincfg.Params) ([]*btcutil.Tx, error) {
	var matchingTxs []*btcutil.Tx

	for _, tx := range block.Transactions {
		if len(tx.TxOut) < 2 {
			continue
		}

		// Check output #1
		output1 := tx.TxOut[1]
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(output1.PkScript, chainParams)
		if err != nil || len(addresses) == 0 || addresses[0].EncodeAddress() != targetAddress.EncodeAddress() || output1.Value == 0 {
			continue
		}

		// Check output #2
		output2 := tx.TxOut[2]
		if output2.Value != 0 || !txscript.IsNullData(output2.PkScript) {
			continue
		}

		// If criteria match, add to the result slice
		matchingTxs = append(matchingTxs, btcutil.NewTx(tx))
	}

	return matchingTxs, nil
}

// Get the UTXO(s) of an address.
// Notice: You need to turn on option -index
// Notice: This is not very accurate, btc nodes tend to forget to track.
// Notice: This won't scale well once the query goes very large.
// Notice: You fill in either P2PKH or P2WPKH address, the result is specific to that address type.
func (r *RpcClient) GetUtxoList(myAddress btcutil.Address) ([]utxo.UTXO, error) {
	// Get the list of unspent transaction outputs
	unspentOutputs, err := r.client.ListUnspentMinMaxAddresses(1, 9999999, []btcutil.Address{myAddress})
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
		}
		if txscript.IsPayToWitnessPubKeyHash(outputPoint.PkScript) {
			pkType = utxo.P2WPKH_SCRIPT_T
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
