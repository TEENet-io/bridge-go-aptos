package btcman

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
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

// Get the UTXO(s) of an address.
// Notice: This is not very accurate, nodes tend to forget to track.
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
