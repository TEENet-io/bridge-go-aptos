package assembler

import (
	"testing"

	"github.com/TEENet-io/bridge-go/multisig_client"
)

const (
	REGTEST_P2TR_ADDR = "bcrt1qvnm6etkkgmyj425hmtdmu4h82zpxudvya4y24n"
	REGTEST_P2TR_PRIV = "cUcHsdBfXphhqLayGuxULxJeABDX74kMtL2gdfyUMVeke3ZJsKQ6"
)

func TestLocalSchnorrOperatorRegtest(t *testing.T) {

	lss, err := multisig_client.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("Cannot create LocalSchnorrSigner: err=%v", err)
	}

	so, err := NewSchnorrOperator(lss, GetRegtestParams())
	if err != nil {
		t.Fatalf("Cannot create SchnorrOperator: err=%v", err)
	}
	t.Logf("P2TR address: %v", so.P2TR.EncodeAddress())

	_addr, err := DecodeAddress(so.P2TR.EncodeAddress(), GetRegtestParams())
	if err != nil {
		t.Fatalf("Cannot properly decode generated address: err=%v, addr=%s", err, _addr)
	}
}

func TestLocalSchnorrOperatorTestnet(t *testing.T) {

	lss, err := multisig_client.NewRandomLocalSchnorrSigner()
	if err != nil {
		t.Fatalf("Cannot create LocalSchnorrSigner: err=%v", err)
	}

	so, err := NewSchnorrOperator(lss, GetTestnetParams())
	if err != nil {
		t.Fatalf("Cannot create SchnorrOperator: err=%v", err)
	}
	t.Logf("P2TR address: %v", so.P2TR.EncodeAddress())

	_addr, err := DecodeAddress(so.P2TR.EncodeAddress(), GetTestnetParams())
	if err != nil {
		t.Fatalf("Cannot properly decode generated address: err=%v, addr=%s", err, _addr)
	}
}

func TestLocalSchnorrOperator2(t *testing.T) {
	priv_key_wif, err := DecodeWIF(REGTEST_P2TR_PRIV)
	if err != nil {
		t.Fatalf("Cannot decode WIF: err=%v", err)
	}
	lss, err := multisig_client.NewLocalSchnorrSigner(priv_key_wif.PrivKey.Serialize())
	if err != nil {
		t.Fatalf("Cannot create LocalSchnorrSigner: err=%v", err)
	}

	so, err := NewSchnorrOperator(lss, GetRegtestParams())
	if err != nil {
		t.Fatalf("Cannot create SchnorrOperator: err=%v", err)
	}
	t.Logf("P2TR address: %v", so.P2TR.EncodeAddress())
}
