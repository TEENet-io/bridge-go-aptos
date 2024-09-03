package wallet

// This file implements interface of a wallet

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"teenet.io/bridge-go/btc/data"
)

// Basic single priv key wallet
// variaous priv key format see README.
type BasicWallet struct {
	ChainConfig *chaincfg.Params // which BTC chain it is on. (mainnet, testnet, regtest)
	PrivKey     *btcutil.WIF
	PubKey      *secp256k1.PublicKey
}

func NewBasicWallet(priv_key_str string, chain_config *chaincfg.Params) (*BasicWallet, error) {
	priv_key, err := btcutil.DecodeWIF(priv_key_str)
	if err != nil {
		return nil, err
	}
	pub_key := priv_key.PrivKey.PubKey()
	return &BasicWallet{chain_config, priv_key, pub_key}, nil
}

func (bw *BasicWallet) AppendOutputP2PKH(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, bw.ChainConfig)
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

func (bw *BasicWallet) AppendOutputP2WPKH(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, bw.ChainConfig)
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

func (bw *BasicWallet) AppendOutputPayToAddress(tx *wire.MsgTx, dst_addr string, amount int64) (*wire.MsgTx, error) {
	btcDstAddress, err := btcutil.DecodeAddress(dst_addr, bw.ChainConfig)
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

// LegacyWallet receives funds via a legacy address.
// It can combine inputs (legacy) and send out to
// both P2PKH & P2WPKH receivers specified in Locking interface.
type LegacyWallet struct {
	BasicWallet
	P2PKH *btcutil.AddressPubKeyHash // legacy address, call .encodeAddress() to get human readable hex represented address
}

func NewLegacyWallet(bw BasicWallet) (*LegacyWallet, error) {
	// Recover a P2PKH address
	p2pkhAddr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(bw.PubKey.SerializeCompressed()), bw.ChainConfig)
	if err != nil {
		return nil, err
	}
	return &LegacyWallet{bw, p2pkhAddr}, nil
}

// Unlock operation generates the tx's inputs secion,
// with every previous output, make a SignatureScript (to unlock it).
// Warning: You should generate Locking Scripts (outputs) firstly on tx,
// then call this function to generate the inputs.
func (lw *LegacyWallet) Unlock(tx *wire.MsgTx, prevOutputs []data.UTXO) (*wire.MsgTx, error) {
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
		if item.PkScriptT == data.P2PKH_SCRIPT_T {
			script, err := txscript.SignatureScript(tx, idx, item.PkScript, txscript.SigHashAll, lw.PrivKey.PrivKey, true)
			if err != nil {
				return nil, err
			}
			tx.TxIn[idx].SignatureScript = script
		} else {
			return nil, fmt.Errorf("UTXO[%d] is not P2PKH script, cannot unlock", idx)
		}
	}
	return tx, nil
}

// Create locking scripts.
// This type of locking sends funds to dst_addr and keep the change to change_addr.
// The change_amount is implied by:
// sum(utxo) = dst_amount + fee_amount + change_amount
func (lw *LegacyWallet) LockByTransfer(
	tx *wire.MsgTx,
	prevOutputs []data.UTXO, // UTXO(s) to spend from.
	dst_addr string, // receiver
	dst_amount int64, // btc amount to receiver in satoshi
	change_addr string, // receiver to receive the change
	fee_amount int64, // amount of mining fee in satoshi
) (*wire.MsgTx, error) {
	var sum int64
	for _, item := range prevOutputs {
		sum += item.Amount
	}
	// Calc change_amount
	change_amount := sum - dst_amount - fee_amount
	if change_amount < 0 {
		return nil, errors.New("change_amount < 0")
	}

	// 1st output: to the dst receiver
	tx, err := lw.AppendOutputPayToAddress(tx, dst_addr, dst_amount)
	if err != nil {
		return nil, err
	}

	// 2nd output: to the change receiver (if change > 0)
	// if change == 0 no need to add this clause.
	if change_amount > 0 {
		tx, err = lw.AppendOutputPayToAddress(tx, change_addr, change_amount)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// MakeTransferTx, make a tx that transfer bitcoin to dst_addr.
// After deduction of fee, keep the change to change_addr.
// You need to send the Tx later via PRC.
func (lw *LegacyWallet) MakeTransferTx(
	dst_addr string,
	dst_amount int64,
	change_addr string,
	fee_amount int64,
	prevOutputs []data.UTXO,
) (*wire.MsgTx, error) {
	// Create a new transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Stuff the locking scripts first.
	tx, err := lw.LockByTransfer(
		tx,
		prevOutputs,
		dst_addr,
		dst_amount,
		change_addr,
		fee_amount,
	)
	if err != nil {
		return nil, err
	}

	// Stuff the unlocking scripts, secondly.
	// Calculate & sign the Tx inputs (by unlocking previous outputs we received)
	tx, err = lw.Unlock(tx, prevOutputs)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// Create locking scripts of BTC2EVM deposit.
// Output #1 to bridge BTC wallet address, with BTC value.
// Output #2 to bridge BTC wallet address, with 0 value and a data piece of OP_RETURN.
// Output #3 to the change address, with remainder BTC value.
func (lw *LegacyWallet) LockByBridgeDeposit(
	tx *wire.MsgTx,
	prevOutputs []data.UTXO,
	btc_bridge_address string, // bridge wallet address on BTC (either P2PKH or P2WPKH type)
	btc_bridge_amount int64, // amount to send to the bridge on BTC (in satoshi)
	fee_amount int64, // amount of mining fee (in satoshi)
	btc_change_address string, // address to receive the change.
	evm_addr string, // EVM receiver's account address
	evm_chain_id int, // EVM chain ID
) (*wire.MsgTx, error) {
	var sum int64
	for _, item := range prevOutputs {
		sum += item.Amount
	}
	// Calc change_amount
	change_amount := sum - btc_bridge_amount - fee_amount
	if change_amount < 0 {
		return nil, errors.New("change_amount < 0")
	}

	// Output #1, correct amount to our btc_bridge_address
	tx, err := lw.AppendOutputPayToAddress(tx, btc_bridge_address, btc_bridge_amount)
	if err != nil {
		return nil, err
	}

	// Output #2, OP_RETURN
	opReturnData, err := data.MakeOpReturnData(evm_chain_id, evm_addr)
	if err != nil {
		return nil, err
	}
	opReturnScript, err := txscript.NullDataScript(opReturnData)
	if err != nil {
		return nil, err
	}
	txOut2 := wire.NewTxOut(0, opReturnScript) // No value for OP_RETURN
	tx.AddTxOut(txOut2)

	// Output #3, remainder btc send to change receiver
	if change_amount > 0 {
		tx, err = lw.AppendOutputPayToAddress(tx, btc_change_address, change_amount)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// Make a Bridge Deposit Tx (BTC2EVM).
// This function is run by the user to deposit BTC to our bridge.
// The user needs to call PRC to send this raw Tx later.
func (lw *LegacyWallet) MakeBridgeDepositTx(
	prevOutputs []data.UTXO,
	btc_bridge_address string, // bridge wallet address on BTC (either P2PKH or P2WPKH type)
	btc_bridge_amount int64, // amount to send to the bridge on BTC (in satoshi)
	fee_amount int64, // amount of mining fee (in satoshi)
	btc_change_address string, // address to receive the change.
	evm_addr string, // EVM receiver's account address
	evm_chain_id int, // EVM chain ID
) (*wire.MsgTx, error) {
	// Create a new transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Stuff the locking scripts first.
	tx, err := lw.LockByBridgeDeposit(
		tx,
		prevOutputs,
		btc_bridge_address,
		btc_bridge_amount,
		fee_amount,
		btc_change_address,
		evm_addr,
		evm_chain_id,
	)
	if err != nil {
		return nil, err
	}

	// Stuff the unlocking scripts, secondly.
	// Calculate & sign the Tx inputs (by unlocking previous outputs we received)
	tx, err = lw.Unlock(tx, prevOutputs)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// SegWitWallet receives funds via a segwit address.
// It can combine inputs (segwit) and send out to
// both P2PKH & P2WPKH receivers specified in Locking interface.
type SegWitWallet struct {
	BasicWallet
	P2WPKH *btcutil.AddressWitnessPubKeyHash // Native segwit address, .encodeAddress() to get human readable hex represented address
}

func NewSegWitWallet(bw BasicWallet) (*SegWitWallet, error) {
	// Recover a P2WPKH (Bech32) address
	p2wpkhAddr, err := btcutil.NewAddressWitnessPubKeyHash(btcutil.Hash160(bw.PubKey.SerializeCompressed()), bw.ChainConfig)
	if err != nil {
		return nil, err
	}
	return &SegWitWallet{bw, p2wpkhAddr}, nil
}

// witness, err := txscript.WitnessSignature(tx, &txscript.TxSigHashes{}, input_idx, int64(item.Amount))
