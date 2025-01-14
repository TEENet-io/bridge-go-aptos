package multisig

import (
	"math/big"
)

// Convert a uncompressed public key to x and y coordinates.
// The public key should be 65 bytes uncompressed [0x04 + x (32byte) + y (32byte)].
// You can furthr convert x and y to big.Int using BytesToBigInt.
func UncompressedToXY(pubKey []byte) ([]byte, []byte) {
	return pubKey[1:33], pubKey[33:]
}

// Convert a x and y coordinates to uncompressed public key.
// You can use BigIntToBytes() to convert X and Y (if in format big.Int) to bytes for the input.
func XYToUncompressed(x, y []byte) []byte {
	return append([]byte{0x04}, append(x, y...)...)
}

// Convert byte slice to big.Int.
func BytesToBigInt(b []byte) *big.Int {
	return new(big.Int).SetBytes(b)
}

// Convert big.Int to byte slice.
func BigIntToBytes(i *big.Int) []byte {
	return i.Bytes()
}
