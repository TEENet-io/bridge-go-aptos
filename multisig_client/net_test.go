package multisig_client

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

var testConfig = ConnectorConfig{
	UserID:        0,
	Name:          "client0",
	Cert:          "config/data/client0.crt",
	Key:           "config/data/client0.key",
	CaCert:        "config/data/client0-ca.crt",
	ServerAddress: "52.184.81.32:6001",
	ServerCACert:  "config/data/node0-ca.crt",
}

func setup() (*Connector, error) {
	if _, err := os.Stat(testConfig.Cert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(testConfig.Key); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(testConfig.CaCert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(testConfig.ServerCACert); os.IsNotExist(err) {
		return nil, err
	}
	c, err := NewConnector(&testConfig)
	return c, err
}

func TestGetPubKey(t *testing.T) {
	c, err := setup()
	if err != nil {
		t.Fatalf("Error setting up connector: %v", err)
	}
	defer c.Close()

	result, err := c.GetPubKey()
	if err != nil {
		t.Fatalf("Error getting public key: %v", err)
	}

	if len(result) != 64 {
		t.Fatalf("Invalid public key length: %d, should be 64 bytes", len(result))
	}
	fmt.Printf("Group Public Key: %x\n", result)
}

func TestGetSignature(t *testing.T) {
	c, err := setup()
	if err != nil {
		t.Fatalf("Error setting up connector: %v", err)
	}
	defer c.Close()

	msgHash := make([]byte, 32)
	_, err = rand.Read(msgHash)
	if err != nil {
		t.Fatalf("Error generating random message hash: %v", err)
	}
	signature, err := c.GetSignature(msgHash)
	if err != nil {
		t.Fatalf("Error getting signature: %v", err)
	}
	if len(msgHash) != 32 {
		t.Fatalf("Invalid message hash length: %d, should be 32 bytes", len(msgHash))
	}
	if len(signature) != 64 {
		t.Fatalf("Invalid signature length: %d, should be 64 bytes", len(signature))
	}
	fmt.Printf("msgHash: %x, Signature: %x\n", msgHash, signature)
}

func TestSignAndVerify(t *testing.T) {

	// message hash to be signed.
	msgHash := make([]byte, 32)
	_, err := rand.Read(msgHash)
	if err != nil {
		t.Fatalf("Error generating random message hash: %v", err)
	}

	c, err := setup()
	if err != nil {
		t.Fatalf("Error setting up connector: %v", err)
	}
	defer c.Close()

	pubkeyBytes, err := c.GetPubKey()
	if err != nil {
		t.Fatalf("Error getting public key: %v", err)
	}

	if len(pubkeyBytes) != 64 {
		t.Fatalf("Invalid public key length: %d, should be 64 bytes", len(pubkeyBytes))
	}
	fmt.Printf("Group Public Key: %x\n", pubkeyBytes)

	sigBytes, err := c.GetSignature(msgHash)
	if err != nil {
		t.Fatalf("Error getting signature: %v", err)
	}
	if len(msgHash) != 32 {
		t.Fatalf("Invalid message hash length: %d, should be 32 bytes", len(msgHash))
	}
	if len(sigBytes) != 64 {
		t.Fatalf("Invalid signature length: %d, should be 64 bytes", len(sigBytes))
	}
	fmt.Printf("msgHash: %x, Signature: %x\n", msgHash, sigBytes)

	// Verify the signature, message, pubkey
	pubkeyBytes = append([]byte{0x04}, pubkeyBytes...)
	pubkey, err := btcec.ParsePubKey(pubkeyBytes)
	if err != nil {
		t.Fatalf("Error parsing public key: %v", err)
	}
	sig, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		t.Fatalf("Error parsing signature: %v", err)
	}

	success := sig.Verify(msgHash, pubkey)
	if !success {
		t.Fatalf("Error verifying signature")
	}
}
