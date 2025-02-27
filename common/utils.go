package common

import (
	"crypto/rand"
	"math/big"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

// HexStrToEthAddress converts a hex string (with/without prefix 0x) to [32]byte
func HexStrToBytes32(hexStr string) [32]byte {
	var bytes32 [32]byte
	copy(bytes32[:], ethcommon.Hex2BytesFixed(TrimHexPrefix(hexStr), 32))
	return bytes32
}

// HexStrToBigInt converts a hex string (with/without prefix 0x) to *big.Int
func HexStrToBigInt(hexStr string) *big.Int {
	bigInt, ok := new(big.Int).SetString(TrimHexPrefix(hexStr), 16)
	if !ok {
		return nil
	}
	return bigInt
}

// BigInt2Bytes32 converts a big int to [32]byte
func BigInt2Bytes32(bigInt *big.Int) [32]byte {
	return [32]byte(ethcommon.LeftPadBytes(bigInt.Bytes(), 32))
}

// BigIntToHexStr converts a big int to hex string with prefix 0x
func BigIntToHexStr(bigInt *big.Int) string {
	return appendHexPrefix(bigInt.Text(16))
}

// Trim 0x or 0X prefix off the string.
func TrimHexPrefix(str string) string {
	s := strings.TrimPrefix(str, "0x")
	return strings.TrimPrefix(s, "0X")
}

func appendHexPrefix(str string) string {
	if strings.HasPrefix(str, "0x") || strings.HasPrefix(str, "0X") {
		return str
	}
	return "0x" + str
}

// RandBytes32 generates [32]byte with random values
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

func RandBytes(n int) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil
	}
	return b
}

func RandBigInt(byteNum int) *big.Int {
	b := RandBytes(byteNum)
	return new(big.Int).SetBytes(b)
}

// Shorten shortens a hex string so that both sides have n characters and
// the rest is replaced with "..."
func Shorten(hexStr string, n int) string {
	str := TrimHexPrefix(hexStr)

	if len(str) <= n*2 {
		return appendHexPrefix(str)
	}
	return appendHexPrefix(str[:n] + "..." + hexStr[len(str)-n:])
}

func BigIntClone(bigInt *big.Int) *big.Int {
	return new(big.Int).Set(bigInt)
}
