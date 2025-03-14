package assembler

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
)

const (
	p1_legacy_priv_key_str = "cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH"
	p1_legacy_addr_str     = "mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT"

	p2_legacy_priv_key_str = "cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY"
	p2_legacy_addr_str     = "moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn"
)

func TestNativeSigner(t *testing.T) {
	priv_key_str := p2_legacy_priv_key_str

	_, err := NewNativeSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create NativeSigner from private key %s", priv_key_str)
	}
}
func TestNativeOperatorP1(t *testing.T) {
	priv_key_str := p1_legacy_priv_key_str

	bs, err := NewNativeSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create NativeSigner from private key %s", priv_key_str)
	}

	ls, err := NewNativeOperator(*bs)
	if err != nil {
		t.Fatalf("Cannot create NativeSigner from private key %s", priv_key_str)
	}

	if ls.P2PKH.EncodeAddress() != p1_legacy_addr_str {
		t.Fatalf("NativeSigner address is not correct, have %s, want %s", ls.P2PKH.EncodeAddress(), p1_legacy_addr_str)
	}
}

func TestNativeOperatorP2(t *testing.T) {
	priv_key_str := p2_legacy_priv_key_str

	bs, err := NewNativeSigner(priv_key_str, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Cannot create NativeSigner from private key %s", priv_key_str)
	}

	ls, err := NewNativeOperator(*bs)
	if err != nil {
		t.Fatalf("Cannot create NativeSigner from private key %s", priv_key_str)
	}

	if ls.P2PKH.EncodeAddress() != p2_legacy_addr_str {
		t.Fatalf("NativeSigner address is not correct, have %s, want %s", ls.P2PKH.EncodeAddress(), p2_legacy_addr_str)
	}
}
