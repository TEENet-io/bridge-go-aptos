package multisig_client

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// Convert a uncompressed public key to x and y coordinates.
// The public key should be 65 bytes uncompressed [0x04 + x (32byte) + y (32byte)].
// You can furthr convert x and y to big.Int using BytesToBigInt.
func UncompressedToXY(pubKey []byte) ([]byte, []byte, error) {
	if len(pubKey) != 65 {
		return nil, nil, fmt.Errorf("public key must be 65 bytes long")
	}
	return pubKey[1:33], pubKey[33:], nil
}

// Convert a x and y coordinates to uncompressed public key.
// You can use BigIntToBytes() to convert X and Y (if in format big.Int) to bytes for the input.
func XYToUncompressed(x, y []byte) ([]byte, error) {
	if len(x) != 32 || len(y) != 32 {
		return nil, fmt.Errorf("x and y must be 32 bytes long")
	}
	return append([]byte{0x04}, append(x, y...)...), nil
}

// Convert byte slice to big.Int.
func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

// Convert big.Int to byte slice.
func BigIntToBytes(i *big.Int) []byte {
	return i.Bytes()
}

// Ensure the byte slice is 64 bytes long.
func Ensure64Bytes(b []byte) bool {
	return len(b) == 64
}

// Ensure the byte slice is 65 bytes long.
func Ensure65Bytes(b []byte) bool {
	return len(b) == 65
}

// Ensure the byte slice is 32 bytes long.
func Ensure32Bytes(b []byte) bool {
	return len(b) == 32
}

// Convert a btcec.PublicKey to x and y coordinates.
func BtcEcPubKeyToXY(pubKey *btcec.PublicKey) (*big.Int, *big.Int) {
	return pubKey.X(), pubKey.Y()
}

// Signature must be 64 bytes long!!!
// first half is Rx, second half is S.
func ConvertSigToRS(mySig *schnorr.Signature) (*big.Int, *big.Int, error) {
	sig := mySig.Serialize()
	if !Ensure64Bytes(sig) {
		return nil, nil, fmt.Errorf("schnorr signature must be 64 bytes long")
	}
	r := BytesToBigInt(sig[:32])
	s := BytesToBigInt(sig[32:])
	return r, s, nil
}
