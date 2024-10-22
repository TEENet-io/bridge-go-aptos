package common

/*
Defines a BTC2EVM deposit data that user shall send along with btc coins to the bridge.
This piece of data contains EVM_CHAIN_ID and EVM_ADDR.
After RLP encoding, the encoded []byte shall be sent as OP_RETURN data in BTC transaction.
See MakeDepositOpReturnData() for more details.
*/

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// BTC to EVM deposit request, contains one output of OP_RETURN data
// It is encoded with RLP.
type DepositData struct {
	EVM_CHAIN_ID [4]byte // Big-endian, most significant is on the left
	EVM_ADDR     [20]byte
}

// Serialize deposit data via RLP
func (dd *DepositData) Serialize() ([]byte, error) {
	return rlp.EncodeToBytes(dd)
}

// util: convert evm_chain_id to [4]byte
func IntToByteArray(n int) ([4]byte, error) {
	// Check for overflow
	if n < 0 || n > 4294967295 {
		return [4]byte{}, errors.New("integer overflow: value must be between 0 and 4294967295")
	}

	// Create a byte array
	var byteArray [4]byte

	// Convert integer to byte array
	byteArray[0] = byte((n >> 24) & 0xFF) // Most significant byte
	byteArray[1] = byte((n >> 16) & 0xFF)
	byteArray[2] = byte((n >> 8) & 0xFF)
	byteArray[3] = byte(n & 0xFF) // least significant byte

	return byteArray, nil
}

// util: convert [4]byte to int
func ByteArrayToInt(b [4]byte) int {
	return int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
}

// util: convert evm_address to [20]byte
func EvmAddrToByteArray(evm_addr string) ([20]byte, error) {
	var r [20]byte
	address := common.HexToAddress(evm_addr)
	byteSlice := address.Bytes()
	if len(byteSlice) != 20 {
		return r, errors.New("cannot convert to 20 bytes")
	}
	copy(r[:], byteSlice)
	return r, nil
}

// util: convert [20]byte to hex string (no "0x" prefix)
func ByteArrayToHexString(b [20]byte) string {
	return common.BytesToAddress(b[:]).Hex()
}

// Util: Create a []byte as an OP_RETURN data via evm chain id and receiver's address
func MakeDepositOpReturnData(evm_chain_id int, evm_addr string) ([]byte, error) {
	evmId, err := IntToByteArray(evm_chain_id)
	if err != nil {
		return nil, err
	}

	evmAddr, err := EvmAddrToByteArray(evm_addr)
	if err != nil {
		return nil, err
	}

	dd := DepositData{evmId, evmAddr}
	return dd.Serialize()
}

// DecodeOpReturnData deposit data from RLP encoded []byte
// TODO: OP_RETURN data may not be handled properly.
func DecodeOpReturnData(data []byte) (*DepositData, error) {
	var dd DepositData
	err := rlp.DecodeBytes(data, &dd)
	if err != nil {
		return nil, err
	}
	return &dd, nil
}
