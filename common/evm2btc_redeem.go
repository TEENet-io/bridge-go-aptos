package common

/*
This file defines a EVM2BTC redeem data that stored within the OP_RETURN
*/

// Currently this piece of data is the requestTxHash (EVM) from Redeem structure and is of 32 bytes.
type RedeemData [32]byte

// Serialize deposit data via RLP
func (rd *RedeemData) Serialize() ([]byte, error) {
	return rd[:], nil
}

func MakeRedeemOpReturnData(rd RedeemData) ([]byte, error) {
	return rd.Serialize()
}
