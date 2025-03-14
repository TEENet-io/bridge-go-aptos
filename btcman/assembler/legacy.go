package assembler

// This file implements interface of a wallet

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
)

// Basic single private key signer
// various private key formats see README.
type BasicSigner struct {
	ChainConfig *chaincfg.Params  // which BTC chain it is on. (mainnet, testnet, regtest)
	PrivKey     *btcec.PrivateKey // private key
	PubKey      *btcec.PublicKey  // public key accordingly
}

// Recover a basic signer from
// private key string (aka wallet-import-format, WIF)
// This is the standard private key string that bitcoin-core software exports.
func NewBasicSigner(priv_key_wif_str string, chain_config *chaincfg.Params) (*BasicSigner, error) {
	priv_key_wif, err := DecodeWIF(priv_key_wif_str)
	if err != nil {
		return nil, err
	}
	return &BasicSigner{chain_config, priv_key_wif.PrivKey, priv_key_wif.PrivKey.PubKey()}, nil
}

// LegacyOperator receives funds via a legacy address (P2PKH).
// It can combine inputs and can send out to
// both P2PKH & P2WPKH receivers.
type LegacyOperator struct {
	BasicSigner
	P2PKH *btcutil.AddressPubKeyHash // legacy address, call .encodeAddress() to get human readable hex represented address
}

func NewLegacyOperator(bw BasicSigner) (*LegacyOperator, error) {
	// Convert Public Key to a P2PKH address
	p2pkhAddr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(bw.PubKey.SerializeCompressed()), bw.ChainConfig)
	if err != nil {
		return nil, err
	}
	return &LegacyOperator{bw, p2pkhAddr}, nil
}

// Unlock operation generates the tx's inputs secion,
// with every previous output, make a SignatureScript (to unlock it).
// Warning: You should generate Locking Scripts (outputs) firstly on tx,
// then call this function to generate the inputs.
func (lo *LegacyOperator) Unlock(tx *wire.MsgTx, prevOutputs []*utxo.UTXO) (*wire.MsgTx, error) {
	// Trick:
	// Both tx.TxIn[] and tx.TxOut[] shall be ready, then you can create sign scripts.
	// If they are not ready the sign will create wrong signature (won't pass the validation of node)
	// In following step the signature script is filled with nil
	for _, item := range prevOutputs {
		txIn := wire.NewTxIn(wire.NewOutPoint(item.TxHash, item.Vout), nil, nil)
		tx.AddTxIn(txIn)
	}
	// In following step signature script is filled with real stuff.
	for idx, item := range prevOutputs {
		script, err := txscript.SignatureScript(tx, idx, item.PkScript, txscript.SigHashAll, lo.PrivKey, true)
		if err != nil {
			return nil, err
		}
		tx.TxIn[idx].SignatureScript = script
	}
	return tx, nil
}

func (lo *LegacyOperator) AppendPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	// Use a common function that already implemented.
	return AppendPayToAddress(tx, dst_chain_cfg, dst_addr, amount)
}
