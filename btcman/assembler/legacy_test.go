package assembler

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

func TestBasicSigner(t *testing.T) {
	priv_key_str := "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"

	_, err := NewBasicSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create wallet from private key %s", priv_key_str)
	}
}
