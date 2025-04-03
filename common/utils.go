package common

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

// The returned string has No 0x prefix
func ByteSliceToPureHexStr(b []byte) string {
	return Trim0xPrefix(ethcommon.Bytes2Hex(b))
}

func HexStrToByteSlice(hexStr string) []byte {
	return ethcommon.Hex2Bytes(Trim0xPrefix(hexStr))
}

// HexStrToEthAddress converts a hex string (with/without prefix 0x) to [32]byte
func HexStrToBytes32(hexStr string) [32]byte {
	var bytes32 [32]byte
	copy(bytes32[:], ethcommon.Hex2BytesFixed(Trim0xPrefix(hexStr), 32))
	return bytes32
}

// HexStrToHash converts a hex string to ethcommon.Hash
func HexStrToHash(hexStr string) ethcommon.Hash {
	return ethcommon.HexToHash(hexStr)
}

// ArrayHexStrToHashes converts an array of hex strings to ethcommon.Hash array
func ArrayHexStrToHashes(hexStrs []string) []ethcommon.Hash {
	hashes := make([]ethcommon.Hash, len(hexStrs))
	for i, hexStr := range hexStrs {
		hashes[i] = HexStrToHash(hexStr)
	}
	return hashes
}

// HexStrToBigInt converts a hex string (with/without prefix 0x) to *big.Int
func HexStrToBigInt(hexStr string) *big.Int {
	bigInt, ok := new(big.Int).SetString(Trim0xPrefix(hexStr), 16)
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
	return Prepend0xPrefix(bigInt.Text(16))
}

// Trim 0x or 0X prefix off the string.
func Trim0xPrefix(str string) string {
	s := strings.TrimPrefix(str, "0x")
	return strings.TrimPrefix(s, "0X")
}

func Prepend0xPrefix(str string) string {
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
	str := Trim0xPrefix(hexStr)

	if len(str) <= n*2 {
		return Prepend0xPrefix(str)
	}
	return Prepend0xPrefix(str[:n] + "..." + hexStr[len(str)-n:])
}

func BigIntClone(bigInt *big.Int) *big.Int {
	return new(big.Int).Set(bigInt)
}

func CompareSlices(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func IsHexChar(c rune) bool {
	match, _ := regexp.MatchString(`[a-fA-F0-9]`, string(c))
	return match
}

// EnsureSafeHexString ensures that the hex string is safe to use
// It can contain 0x as prefix or not.
// It can contain a-f, A-F, 0-9
// It doesn't contain any other characters
func EnsureSafeAddressHexString(hexStr string) bool {
	if len(hexStr) < 2 {
		return false
	}
	if len(hexStr) > 100 {
		return false
	}
	if strings.HasPrefix(hexStr, "0x") || strings.HasPrefix(hexStr, "0X") {
		hexStr = hexStr[2:]
	}
	for _, c := range hexStr {
		// use regex to match each charater of the string
		if !IsHexChar(c) {
			return false
		}
	}
	return true
}
