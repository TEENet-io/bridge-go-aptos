package etherman

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

func HexStrToBytes32(hexStr string) [32]byte {
	var bytes32 [32]byte
	copy(bytes32[:], common.Hex2BytesFixed(strings.TrimPrefix(hexStr, "0x"), 32))
	return bytes32
}

func HexStrToBigInt(hexStr string) *big.Int {
	bigInt, ok := new(big.Int).SetString(strings.TrimPrefix(hexStr, "0x"), 16)
	if !ok {
		return nil
	}
	return bigInt
}

func Sign(sk *btcec.PrivateKey, msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(sk, msg[:])
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}

func EncodePacked(values ...interface{}) []byte {
	var res [][]byte
	for _, value := range values {
		switch v := value.(type) {
		case string:
			res = append(res, EncodeString(v))
		case []byte:
			res = append(res, v)
		case *big.Int:
			res = append(res, math.U256Bytes(v))
		case []string:
			res = append(res, EncodeStringArray(v))
		case []*big.Int:
			res = append(res, EncodeBigIntArray(v))
		}
	}
	return bytes.Join(res, nil)
}

func EncodeBigIntArray(arr []*big.Int) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, math.U256Bytes(v))
	}

	return bytes.Join(res, nil)
}

func EncodeStringArray(arr []string) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, EncodeString(v))
	}

	return bytes.Join(res, nil)
}

func EncodeString(v string) []byte {
	if strings.HasPrefix(v, "0x") {
		return EncodeHexString(v)
	}

	return EncodeRawString(v)
}

func EncodeRawString(v string) []byte {
	return []byte(v)
}

func EncodeHexString(v string) []byte {
	decoded, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
	if err != nil {
		panic(err)
	}
	return decoded
}

func EncodeUint256(v string) []byte {
	bn := new(big.Int)
	bn.SetString(v, 10)
	return math.U256Bytes(bn)
}

func EncodeUint256Array(arr []string) []byte {
	var res [][]byte
	for _, v := range arr {
		b := EncodeUint256(v)
		res = append(res, b)
	}

	return bytes.Join(res, nil)
}
