package etherman

import (
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/ethereum/go-ethereum/common"
)

func HexStrToBytes32(hexStr string) [32]byte {
	var bytes32 [32]byte
	copy(bytes32[:], common.Hex2BytesFixed(TrimHexPrefix(hexStr), 32))
	return bytes32
}

func HexStrToBigInt(hexStr string) *big.Int {
	bigInt, ok := new(big.Int).SetString(TrimHexPrefix(hexStr), 16)
	if !ok {
		return nil
	}
	return bigInt
}

func TrimHexPrefix(str string) string {
	s := strings.TrimPrefix(str, "0x")
	return strings.TrimPrefix(s, "0X")
}

func Sign(sk *btcec.PrivateKey, msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(sk, msg[:])
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}

func getMapKeysValues[K comparable, V comparable](m map[K]V) (keys []K, values []V) {
	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}
	return
}
