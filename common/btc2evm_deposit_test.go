package common

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// evm mainnet = 1
	evm_chain_id = [4]byte{0, 0, 0, 1}
	// evm addr = 0x95222290DD7278Aa3Ddd389Cc1E1d165CC4BAfe5
	evm_addr, _ = hex.DecodeString("95222290dd7278aa3ddd389cc1e1d165cc4bafe5")
)

func TestDepositData(t *testing.T) {
	// slice -> array
	var real_evm_addr [20]byte
	copy(real_evm_addr[:], evm_addr)

	dd := DepositData{evm_chain_id, real_evm_addr}

	encoded, err := dd.Serialize()
	if err != nil {
		t.Fatalf(`RLP encode failed, evm_chain_id %x, evm_addr %x`, evm_chain_id, evm_addr)
	}

	// Compare the struct encode result
	// with pure [id []byte, addr []byte] encode result
	var original [][]byte
	original = append(original, evm_chain_id[:])
	original = append(original, evm_addr)
	compare, err := rlp.EncodeToBytes(original)
	if err != nil {
		t.Fatalf(`RLP encode failed of comparison group %v`, original)
	}

	if !reflect.DeepEqual(encoded, compare) {
		t.Fatalf(`RLP of struct DepositData and pure array not match`)
	}

	// Make sure the encoded is 27 bytes total
	if len(encoded) != 27 {
		t.Fatalf("RLP encoded result is not %d bytes", 27)
	}
}

func TestDecodeDeposit(t *testing.T) {
	_hex_data := "da8400aa36a79434c7dfb77d536e9698b8d6be86c339e460026827"
	_byte_data, _ := hex.DecodeString(_hex_data)

	var dd DepositData
	err := rlp.DecodeBytes(_byte_data, &dd)
	if err != nil {
		t.Fatalf("RLP decode failed %v", err)
	}

	_suppose_addr := "0x34c7dFB77D536e9698b8d6BE86c339e460026827"
	if _suppose_addr != ByteArrayToHexString(dd.EVM_ADDR) {
		t.Fatalf("Decoded address not match, expected %s, got %s", _suppose_addr, ByteArrayToHexString(dd.EVM_ADDR))
	}

	_suppose_chain_id := 11155111
	if _suppose_chain_id != ByteArrayToInt(dd.EVM_CHAIN_ID) {
		t.Fatalf("Decoded chain_id not match, expected %d, got %d", _suppose_chain_id, ByteArrayToInt(dd.EVM_CHAIN_ID))
	}
}
