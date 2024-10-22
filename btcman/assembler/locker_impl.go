package assembler

/*
This file implements BTC "Locker" interface.

Since locking scripts do not require any prior knowledge of private keys,
it is universal to all wallet implementations.

So we can do it here.
*/

import (
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func AddP2PKH(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, dst_chain_cfg)
	if err != nil {
		return nil, err
	}
	// Check if dst_addr is really a P2PKH address
	if realAddress, ok := btcDstAddress.(*btcutil.AddressPubKeyHash); ok {
		txOutScript, err := txscript.PayToAddrScript(realAddress) // simple
		if err != nil {
			return nil, err
		}
		txOut := wire.NewTxOut(amount, txOutScript)
		tx.AddTxOut(txOut)
		return tx, nil
	} else {
		return nil, errors.New("%s is not a P2PKH (legacy) address")
	}
}

func AddP2WPKH(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, dst_chain_cfg)
	if err != nil {
		return nil, err
	}
	// Check if dst_addr is really a P2WPKH address
	if realAddress, ok := btcDstAddress.(*btcutil.AddressWitnessPubKeyHash); ok {
		txOutScript, err := txscript.PayToAddrScript(realAddress) // simple
		if err != nil {
			return nil, err
		}
		txOut := wire.NewTxOut(amount, txOutScript)
		tx.AddTxOut(txOut)
		return tx, nil
	} else {
		return nil, errors.New("%s is not a P2WPKH (SegWit) address")
	}
}

func AddPayToAddress(tx *wire.MsgTx, dst_chain_cfg *chaincfg.Params, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, dst_chain_cfg)
	if err != nil {
		return nil, err
	}

	txOutScript, err := txscript.PayToAddrScript(btcDstAddress) // simple
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(amount, txOutScript)
	tx.AddTxOut(txOut)
	return tx, nil
}
