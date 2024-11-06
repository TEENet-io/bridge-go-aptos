package common

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	logger "github.com/sirupsen/logrus"
)

func EncodePacked(values ...interface{}) []byte {
	var res [][]byte
	for _, value := range values {
		switch v := value.(type) {
		case string:
			res = append(res, encodeString(v))
		case []byte:
			res = append(res, v)
		case [32]byte:
			res = append(res, v[:])
		case [][32]byte:
			res = append(res, encodeBytes32Array(v))
		case []string:
			res = append(res, encodeStringArray(v))
		case *big.Int:
			res = append(res, math.U256Bytes(v))
		case []*big.Int:
			res = append(res, encodeBigIntArray(v))
		case common.Hash:
			res = append(res, v[:])
		case []common.Hash:
			res = append(res, encodeHashArray(v))
		case common.Address:
			res = append(res, encodeString(v.String()))
		case []common.Address:
			res = append(res, encodeAddressArray(v))
		}
	}
	return bytes.Join(res, nil)
}

func encodeAddressArray(arr []common.Address) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, encodeString(v.String()))
	}

	return bytes.Join(res, nil)
}

func encodeHashArray(arr []common.Hash) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, v[:])
	}

	return bytes.Join(res, nil)
}

func encodeBytes32Array(arr [][32]byte) []byte {
	var res [][]byte
	for _, v := range arr {
		res = append(res, v[:])
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
		logger.Fatal(err)
	}
	return decoded
}
