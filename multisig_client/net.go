package multisig_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	pb "github.com/TEENet-io/bridge-go/rpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Configuration for the connector client
type ConnectorConfig struct {
	// user id in multisig
	UserID int

	// name given to the user
	Name string

	// path to the TLS certificate and key used to run a TLS client
	Cert string
	Key  string

	// path to the CA certificate used to authenticate the user during TLS handshake
	CaCert string

	// IP address of the remote RPC server, in the form of host:port
	ServerAddress string

	// path to the CA certificate used to authenticate the remote RPC server during TLS handshake
	ServerCACert string
}

// Create a TLS config, from loading the cert, key, and CA cert files (paths)
func createTLSConfig(certFilePath, keyFilePath, serverCaCertFilePath string) (*tls.Config, error) {
	// Load client certificate and key
	cert, err := tls.LoadX509KeyPair(certFilePath, keyFilePath)
	if err != nil {
		fmt.Printf("failed to load client certificate and key: %v", err)
		return nil, err
	}

	// Load CA certificate
	caCertPool := x509.NewCertPool()
	log.Printf("Loading CA cert: %s", serverCaCertFilePath)
	caCert, err := os.ReadFile(serverCaCertFilePath)
	if err != nil {
		fmt.Printf("Failed to read CA certificate. err: %v", err)
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}, nil
}

type Connector struct {
	configuration *ConnectorConfig
	service       pb.SignatureClient // service that can do sign() and getpubkey()
	grpcConn      *grpc.ClientConn   // underlying connection manager.
}

// Create a new connector, given a client configuration
func NewConnector(clientConfig *ConnectorConfig) (*Connector, error) {

	_connector := &Connector{
		configuration: clientConfig,
	}

	// Create a TLS configuration for the client
	tlsConfig, err := createTLSConfig(clientConfig.Cert, clientConfig.Key, clientConfig.ServerCACert)
	if err != nil {
		return nil, fmt.Errorf("error creating TLS config: %v", err)
	}

	// Connect to the RPC server over TLS
	conn, err := grpc.NewClient(
		clientConfig.ServerAddress,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, fmt.Errorf("error connecting to RPC server: %v", err)
	}

	_connector.grpcConn = conn

	// Create an RPC client
	client := pb.NewSignatureClient(conn)

	_connector.service = client

	return _connector, nil
}

// Don't forget to close conn once object is done using.
func (c *Connector) Close() {
	c.grpcConn.Close()
}

// GetPubKey RPC call
func (c *Connector) GetPubKey() ([]byte, error) {
	getPubKeyRequest := &pb.GetPubKeyRequest{UserID: int32(c.configuration.UserID)}
	getPubKeyReply, err := c.service.GetPubKey(context.Background(), getPubKeyRequest)
	if err != nil {
		log.Fatalf("Error calling GetPubKey: %v", err)
	}

	if !getPubKeyReply.GetSuccess() {
		return nil, fmt.Errorf("failed to get pubkey")
	}

	return getPubKeyReply.GetGroupPublicKey(), nil
}

// GetSignature RPC call
func (c *Connector) GetSignature(msg []byte) ([]byte, error) {
	getSignatureRequest := &pb.GetSignatureRequest{Msg: msg}
	getSignatureReply, err := c.service.GetSignature(context.Background(), getSignatureRequest)
	if err != nil {
		log.Fatalf("Error calling GetSignature: %v", err)
	}

	if !getSignatureReply.GetSuccess() {
		return nil, fmt.Errorf("failed to get signature")
	}

	return getSignatureReply.GetSignature(), nil
}
