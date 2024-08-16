package fabric

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/edgeflare/fabric-oidc-proxy/internal/config"
	"github.com/edgeflare/pgo"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GWClient wraps the Fabric Gateway client.
type GWClient struct {
	*client.Gateway
}

// NewGatewayClient creates a new Fabric Gateway client.
// It establishes a gRPC connection to the Fabric peer, creates user identity and signing objects,
// and returns a GWClient instance for interacting with the Fabric network.
func NewGatewayClient(ctx context.Context, conf ...config.Config) (*GWClient, error) {
	// Create a copy of the global configuration to avoid modifying it directly
	localCfg := cfg

	// If a configuration is provided, override MSPCert and MSPKey
	if len(conf) > 0 {
		localCfg.Fabric.GW.MSPCert = conf[0].Fabric.GW.MSPCert
		localCfg.Fabric.GW.MSPKey = conf[0].Fabric.GW.MSPKey
	}

	// Load configuration from the local copy
	clientConn, err := newGrpcConnection(ctx) // Pass the context to newGrpcConnection if needed
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	id, err := newIdentity(localCfg.Fabric.GW.MSPCert, localCfg.Fabric.GW.MSPID)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	sign, err := newSign(localCfg.Fabric.GW.MSPKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	gateway, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithClientConnection(clientConn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to gateway: %w", err)
	}

	return &GWClient{
		Gateway: gateway,
	}, nil
}

// newGrpcConnection creates a new gRPC connection to the Fabric peer.
// It loads the TLS certificate, configures transport credentials, and establishes the connection.
func newGrpcConnection(ctx context.Context) (*grpc.ClientConn, error) {
	_ = ctx
	certificatePEM, err := os.ReadFile(cfg.Fabric.GW.TLSTrustedCerts)
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS certificate file: %w", err)
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, cfg.Fabric.GW.PeerServerNameOverride)

	connection, err := grpc.NewClient(cfg.Fabric.GW.PeerEndpoint,
		grpc.WithTransportCredentials(transportCredentials),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	return connection, nil
}

// SubmitTransaction submits a transaction to the Fabric network.
// It retrieves user information from the context, creates a gateway client using the user's credentials,
// and submits the transaction to the specified channel and chaincode.
func SubmitTransaction(ctx context.Context, channelID, chaincodeID, fn string, args ...string) ([]byte, error) {
	user, ok := ctx.Value(pgo.OIDCUserCtxKey).(*oidc.IntrospectionResponse)
	if !ok || user == nil {
		return nil, fmt.Errorf("no user found")
	}

	userDir := filepath.Join(cfg.Fabric.CA.ClientHome, "users", user.Subject)

	keyPath, err := GetMSPKeyfile(userDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get MSP keyfile: %w", err)
	}

	certPath := filepath.Join(cfg.Fabric.CA.ClientHome, "users", user.Subject, "msp", "signcerts", "cert.pem")

	cfg := config.Config{
		Fabric: config.FabricConfig{
			GW: config.FabricGWConfig{
				MSPCert: certPath,
				MSPKey:  keyPath,
			},
		},
	}

	gw, err := NewGatewayClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway client: %w", err)
	}

	network := gw.GetNetwork(channelID)
	contract := network.GetContract(chaincodeID)

	resultBytes, err := contract.SubmitTransaction(fn, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	return resultBytes, nil
}

// newIdentity creates a new X509 identity for the user.
// It loads the user's certificate and creates an identity object.
func newIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certificatePEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, fmt.Errorf("failed to create X509 identity: %w", err)
	}

	return id, nil
}

// newSign creates a new signing function using the user's private key.
// It loads the private key and creates a signing object.
func newSign(keyPath string) (identity.Sign, error) {
	privateKeyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create sign function: %w", err)
	}

	return sign, nil
}

// parsePrivateKey parses a private key from PEM-encoded bytes.
// It attempts to parse the key in various formats (PKCS8, EC, PKCS1) and returns the parsed key.
func parsePrivateKey(pemBytes []byte) (interface{}, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	return nil, fmt.Errorf("failed to parse private key")
}
