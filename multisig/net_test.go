package multisig

import (
	"fmt"
	"os"
	"testing"
)

var testConfig = ConnectorConfig{
	UserID:        0,
	Name:          "client0",
	Cert:          "config/data/client0.crt",
	Key:           "config/data/client0.key",
	CaCert:        "config/data/client0-ca.crt",
	ServerAddress: "20.205.130.99:6001",
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

	result, err := c.GetSignature([]byte("hello1"))
	if err != nil {
		t.Fatalf("Error getting signature: %v", err)
	}
	if len(result) != 64 {
		t.Fatalf("Invalid signature length: %d, should be 64 bytes", len(result))
	}
	fmt.Printf("Signature: %x\n", result)
}
