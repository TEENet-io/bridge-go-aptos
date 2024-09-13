package common

import (
	"crypto/rand"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func HexStrToBytes32(hexStr string) [32]byte {
	var bytes32 [32]byte
	copy(bytes32[:], common.Hex2BytesFixed(trimHexPrefix(hexStr), 32))
	return bytes32
}

func HexStrToBigInt(hexStr string) *big.Int {
	bigInt, ok := new(big.Int).SetString(trimHexPrefix(hexStr), 16)
	if !ok {
		return nil
	}
	return bigInt
}

func trimHexPrefix(str string) string {
	s := strings.TrimPrefix(str, "0x")
	return strings.TrimPrefix(s, "0X")
}

func appendHexPrefix(str string) string {
	if strings.HasPrefix(str, "0x") || strings.HasPrefix(str, "0X") {
		return str
	}
	return "0x" + str
}

func Bytes32ToHexStr(bytes32 [32]byte) string {
	return appendHexPrefix(common.Bytes2Hex(bytes32[:]))
}

func BigIntToHexStr(bigInt *big.Int) string {
	return appendHexPrefix(bigInt.Text(16))
}

func RandBytes32() [32]byte {
	var b [32]byte
	n, err := rand.Read(b[:])

	if err != nil {
		return [32]byte{}
	}
	if n != 32 {
		return [32]byte{}
	}

	return b
}

func RandAddress() common.Address {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return common.Address{}
	}
	return common.BytesToAddress(b[:])
}
