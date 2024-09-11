package common

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/math"
)

func EncodePacked(values ...interface{}) []byte {
	var res [][]byte
	for _, value := range values {
		switch v := value.(type) {
		case string:
			res = append(res, encodeString(v))
		case []byte:
			res = append(res, v)
		case *big.Int:
			res = append(res, math.U256Bytes(v))
		case []string:
			res = append(res, encodeStringArray(v))
		case []*big.Int:
			res = append(res, encodeBigIntArray(v))
		}
	}
	return bytes.Join(res, nil)
}

func encodeBigIntArray(arr []*big.Int) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, math.U256Bytes(v))
	}

	return bytes.Join(res, nil)
}

func encodeStringArray(arr []string) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, encodeString(v))
	}

	return bytes.Join(res, nil)
}

func encodeString(v string) []byte {
	if strings.HasPrefix(v, "0x") {
		return encodeHexString(v)
	}

	return encodeRawString(v)
}

func encodeRawString(v string) []byte {
	return []byte(v)
}

func encodeHexString(v string) []byte {
	decoded, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
	if err != nil {
		panic(err)
	}
	return decoded
}

func encodeUint256(v string) []byte {
	bn := new(big.Int)
	bn.SetString(v, 10)
	return math.U256Bytes(bn)
}

func encodeUint256Array(arr []string) []byte {
	var res [][]byte
	for _, v := range arr {
		b := encodeUint256(v)
		res = append(res, b)
	}

	return bytes.Join(res, nil)
}
