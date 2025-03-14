package assembler

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/common"
)

type Assembler struct {
	ChainConfig *chaincfg.Params // which BTC chain it is on. (mainnet, testnet, regtest)
	Op          Operator         // can do unlock/locking script on a btc transaction.
}

// Create a locking script on a Tx, to transfer out money to a single receiver.
// This type of locking sends funds to dst_addr and keep the change to change_addr.
// The change_amount is implied by:
// sum(utxo) = dst_amount + fee_amount + change_amount
func (myAss *Assembler) craftTransferOutOutput(
	tx *wire.MsgTx,
	prevOutputs []*utxo.UTXO, // UTXO(s) to spend from.
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
	tx, err := AppendPayToAddress(tx, myAss.ChainConfig, dst_addr, dst_amount)
	if err != nil {
		return nil, err
	}

	// 2nd output: to the change receiver (if change > 0)
	// if change == 0 no need to add this clause.
	if change_amount > 0 {
		tx, err = AppendPayToAddress(tx, myAss.ChainConfig, change_addr, change_amount)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// Make a raw tx that transfer some bitcoin to dst_addr.
// It takes care of both locking + unlocking.
// After deduction of mining fee, keep the change to change_addr.
// You need to send the Tx later via PRC.
func (myAss *Assembler) MakeTransferOutTx(
	dst_addr string,
	dst_amount int64,
	change_addr string,
	fee_amount int64,
	prevOutputs []*utxo.UTXO,
) (*wire.MsgTx, error) {
	// Create a new transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Stuff the locking scripts first.
	tx, err := myAss.craftTransferOutOutput(
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
	tx, err = myAss.Op.Unlock(tx, prevOutputs)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// craftRedeemOutput creates a Redeem (withdraw) Tx.
// output #1, satoshi to the user receiver.
// output #2, redeem data to be sent in OP_RETURN.
// output #3, satoshi to the our change receiver.
func (myAss *Assembler) craftRedeemOutput(
	tx *wire.MsgTx,
	prevOutputs []*utxo.UTXO, // UTXO(s) to spend from.
	dst_addr string, // receiver
	dst_amount int64, // btc amount to receiver in satoshi
	redeemData common.RedeemData, // data to be sent in OP_RETURN
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
		return nil, fmt.Errorf("change_amount < 0, sum: %d, dst_amount: %d, fee_amount: %d", sum, dst_amount, fee_amount)
	}

	// 1st output: to the dst receiver
	tx, err := AppendPayToAddress(tx, myAss.ChainConfig, dst_addr, dst_amount)
	if err != nil {
		return nil, err
	}

	// 2nd output: OP_RETURN data
	// Output #2, OP_RETURN
	opReturnData, err := common.MakeRedeemOpReturnData(redeemData)
	if err != nil {
		return nil, err
	}
	opReturnScript, err := txscript.NullDataScript(opReturnData)
	if err != nil {
		return nil, err
	}
	txOut2 := wire.NewTxOut(0, opReturnScript) // No value for OP_RETURN
	tx.AddTxOut(txOut2)

	// 3rd output: to the change receiver (if change > 0)
	// if change == 0 no need to add this clause.
	if change_amount > 0 {
		tx, err = AppendPayToAddress(tx, myAss.ChainConfig, change_addr, change_amount)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// Make a raw tx that transfer some bitcoin to dst_addr.
// It takes care of both locking + unlocking.
// After deduction of mining fee, keep the change to change_addr.
// You need to send the Tx later via PRC.
func (myAss *Assembler) MakeRedeemTx(
	dst_addr string,
	dst_amount int64,
	redeemData common.RedeemData, // data to be sent in OP_RETURN
	change_addr string,
	fee_amount int64,
	prevOutputs []*utxo.UTXO,
) (*wire.MsgTx, error) {
	// Create a new transaction
	tx := wire.NewMsgTx(wire.TxVersion)

	// Stuff the locking scripts first.
	tx, err := myAss.craftRedeemOutput(
		tx,
		prevOutputs,
		dst_addr,
		dst_amount,
		redeemData,
		change_addr,
		fee_amount,
	)
	if err != nil {
		return nil, err
	}

	// Stuff the unlocking scripts, secondly.
	// Calculate & sign the Tx inputs (by unlocking previous outputs we received)
	tx, err = myAss.Op.Unlock(tx, prevOutputs)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// Create 3 locking scripts on a given Tx.
// These 3 scripts combined is recognized as "BTC2EVM deposit".
// Output #1 to bridge BTC wallet address, with BTC value.
// Output #2 to bridge BTC wallet address, with 0 value and a data piece of OP_RETURN.
// Output #3 to the change address, with remainder BTC value.
func (myAss *Assembler) craftBridgeDepositOutputs(
	tx *wire.MsgTx,
	prevOutputs []*utxo.UTXO,
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
	tx, err := AppendPayToAddress(tx, myAss.ChainConfig, btc_bridge_address, btc_bridge_amount)
	if err != nil {
		return nil, err
	}

	// Output #2, OP_RETURN
	opReturnData, err := common.MakeDepositOpReturnData(evm_chain_id, evm_addr)
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
		tx, err = AppendPayToAddress(tx, myAss.ChainConfig, btc_change_address, change_amount)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// Make a Bridge Deposit Tx (BTC2EVM).
// This function is run by the user to generate a legit deposit Tx of BTC to our bridge.
// The user needs to call PRC to send the raw Tx later.
func (myAss *Assembler) MakeBridgeDepositTx(
	prevOutputs []*utxo.UTXO,
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
	tx, err := myAss.craftBridgeDepositOutputs(
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
	tx, err = myAss.Op.Unlock(tx, prevOutputs)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
