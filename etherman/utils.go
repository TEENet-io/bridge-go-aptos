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

func Sign(sk *btcec.PrivateKey, msg []byte) (*big.Int, *big.Int, error) {
	sig, err := schnorr.Sign(sk, msg[:])
	if err != nil {
		return nil, nil, err
	}

	bytes := sig.Serialize()
	return new(big.Int).SetBytes(bytes[:32]), new(big.Int).SetBytes(bytes[32:]), nil
}
