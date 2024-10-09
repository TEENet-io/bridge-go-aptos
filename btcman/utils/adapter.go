package utils

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/common"
)

// IsDepositTx checks if the given tx is a bridge deposit tx
func IsDepositTx(tx *wire.MsgTx, targetAddress btcutil.Address, chainParams *chaincfg.Params) bool {
	// Check if the tx has at least 2 outputs
	if len(tx.TxOut) < 2 {
		return false
	}

	// Check output #1, if pays to us?
	flag1 := false
	output1 := tx.TxOut[0]
	_, addresses, _, err := txscript.ExtractPkScriptAddrs(output1.PkScript, chainParams)
	if err != nil || len(addresses) == 0 || addresses[0].EncodeAddress() != targetAddress.EncodeAddress() || output1.Value == 0 {
		flag1 = false
	} else {
		flag1 = true
	}

	// Check output #2, if OP_RETURN?
	flag2 := false
	output2 := tx.TxOut[1]
	if output2.Value == 0 && txscript.IsNullData(output2.PkScript) {
		flag2 = true
	} else {
		flag2 = false
	}

	return flag1 && flag2
}

// CraftDepositAction creates a DepositAction from the given tx
func CraftDepositAction(tx *wire.MsgTx, blockHeight int32, block *wire.MsgBlock, targetAddress btcutil.Address, chainParams *chaincfg.Params) (*btcaction.DepositAction, error) {

	// Decode the op_retun data
	output2 := tx.TxOut[1]
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
