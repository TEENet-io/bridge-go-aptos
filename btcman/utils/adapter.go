package utils

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/common"
)

// MaybeDepositTx checks if the given tx is really a bridge deposit tx.
func MaybeDepositTx(tx *wire.MsgTx, targetAddress btcutil.Address, chainParams *chaincfg.Params) bool {
	// Check if the tx has at least 2 outputs
	if len(tx.TxOut) < 2 {
		return false
	}

	// Check output #0, if pays to us?
	flag1 := false
	output1 := tx.TxOut[0]
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(output1.PkScript, chainParams)
	if err != nil || len(addresses) == 0 || addresses[0].EncodeAddress() != targetAddress.EncodeAddress() || output1.Value == 0 {
		flag1 = false
	} else {
		flag1 = true
	}

	// Check output #1, if OP_RETURN?
	flag2 := false
	output2 := tx.TxOut[1]
	if output2.Value == 0 && txscript.IsNullData(output2.PkScript) {
		flag2 = true
	} else {
		flag2 = false
	}

	return flag1 && flag2
}

func MaybeJustTransfer(tx *wire.MsgTx, targetAddress btcutil.Address, chainParams *chaincfg.Params) []struct {
	Vout   int
	Amount int64
} {
	// Check if the tx has at least 1 output
	if len(tx.TxOut) < 1 {
		return nil
	}

	var results []struct {
		Vout   int
		Amount int64
	}

	// for each txout, loop, if any of them pays to us, then it is a transfer.
	// mark the vout.
	for i := 0; i < len(tx.TxOut); i++ {
		output := tx.TxOut[i]
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(output.PkScript, chainParams)
		if err != nil || len(addresses) == 0 || addresses[0].EncodeAddress() != targetAddress.EncodeAddress() || output.Value == 0 {
			continue
		}

		results = append(results, struct {
			Vout   int
			Amount int64
		}{
			Vout:   i,
			Amount: output.Value,
		})
	}

	return results
}

// CraftDepositAction creates a DepositAction from the given tx
func CraftDepositAction(tx *wire.MsgTx, blockHeight int32, block *wire.MsgBlock, targetAddress btcutil.Address, chainParams *chaincfg.Params) (*btcaction.DepositAction, error) {

	// Decode the op_retun data
	output2 := tx.TxOut[1]
	// TODO: OP_RETURN data may not be in good shape
	data, err := common.DecodeOpReturnData(output2.PkScript)
	if err != nil {
		return nil, err
	}

	// Create the deposit action
	deposit := &btcaction.DepositAction{
		Basic: btcaction.Basic{
			BlockNumber: int(blockHeight),
			BlockHash:   block.BlockHash().String(),
			TxHash:      tx.TxHash().String(),
		},
		DepositValue:    tx.TxOut[0].Value,
		DepositReceiver: targetAddress.EncodeAddress(),
		EvmID:           int32(common.ByteArrayToInt(data.EVM_CHAIN_ID)),
		EvmAddr:         common.ByteArrayToHexString(data.EVM_ADDR),
	}

	return deposit, nil
}

// MaybeRedeemTx checks if the given tx may be is a redeem.
func MaybeRedeemTx(tx *wire.MsgTx, changeAddress btcutil.Address, chainParams *chaincfg.Params) bool {
	// Check if the tx has at least 3 outputs
	if len(tx.TxOut) < 3 {
		return false
	}

	// output #0 = pay to user (we don't care)
	// output #1 = OP_RETURN + data
	// output #2 = pay to change (which is us)

	// Check output #1, if OP_RETURN + data?
	flag_op := false
	output_n_1 := tx.TxOut[1]
	if output_n_1.Value == 0 && txscript.IsNullData(output_n_1.PkScript) {
		flag_op = true
	} else {
		flag_op = false
	}

	// Check output #2, if pays to us?
	flag_to_us := false
	output_n_2 := tx.TxOut[2]
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(output_n_2.PkScript, chainParams)
	if err != nil || len(addresses) == 0 || addresses[0].EncodeAddress() != changeAddress.EncodeAddress() || output_n_2.Value == 0 {
		flag_to_us = false
	} else {
		flag_to_us = true
	}

	return flag_op && flag_to_us
}
