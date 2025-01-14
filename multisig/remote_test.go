package multisig

import (
	"os"
	"testing"

	"github.com/TEENet-io/bridge-go/common"
)

var connConfig = ConnectorConfig{
	UserID:        0,
	Name:          "client0",
	Cert:          "config/data/client0.crt",
	Key:           "config/data/client0.key",
	CaCert:        "config/data/client0-ca.crt",
	ServerAddress: "20.205.130.99:6001",
	ServerCACert:  "config/data/node0-ca.crt",
}

func setupConn(connConfig ConnectorConfig) (*Connector, error) {
	if _, err := os.Stat(connConfig.Cert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.Key); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.CaCert); os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Stat(connConfig.ServerCACert); os.IsNotExist(err) {
		return nil, err
	}
	c, err := NewConnector(&connConfig)
	return c, err
}

func TestSignAndVerify2(t *testing.T) {
	c, err := setupConn(connConfig)
	if err != nil {
		t.Fatalf("Error setting up connector: %v", err)
	}
	defer c.Close()

	ss := NewRemoteSchnorrSigner(c)

	message := []byte("hello world")
	rx, s, err := ss.Sign(message)
	if err != nil {
		t.Fatalf("Error signing: %v", err)
	}

	pubKeyX, pubKeyY, err := ss.Pub()
	if err != nil {
		t.Fatalf("Error getting public key: %v", err)
	}

	t.Logf("Group Public Key: (%x, %x)\n", pubKeyX, pubKeyY)

	// We only verify the X-coordinate of the public key
	// according to common.btcuti_test.go
	ok := common.Verify(BigIntToBytes(pubKeyX), message, rx, s)
	if !ok {
		t.Fatalf("Error verifying signature")
	}
}
