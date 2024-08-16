package fabric

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/edgeflare/fabric-oidc-proxy/internal/config"
	"github.com/hyperledger/fabric-ca/api"
	"github.com/hyperledger/fabric-ca/lib"
	"github.com/hyperledger/fabric-ca/lib/client/credential/x509"
	"github.com/hyperledger/fabric-ca/lib/tls"
)

// CAClient wraps the Fabric CA client with additional functionality.
type CAClient struct {
	caClient *lib.Client
}

// MSPKeyCert holds the certificate and key for a user's Membership Service Provider (MSP).
type MSPKeyCert struct {
	Cert string `json:"msp.crt"`
	Key  string `json:"msp.key"`
}

// NewCAClient initializes and returns a new Fabric CA client using configuration from the config package
func NewCAClient(cfg *config.Config, userHomeDir ...string) (*CAClient, error) {
	var homeDir string

	if len(userHomeDir) > 0 {
		homeDir = userHomeDir[0]
	} else {
		homeDir = cfg.Fabric.CA.ClientHome
	}

	caClient := &lib.Client{
		HomeDir: homeDir,
		Config: &lib.ClientConfig{
			URL:    cfg.Fabric.CA.URL,
			MSPDir: cfg.Fabric.CA.ClientMSPDir,
			TLS: tls.ClientTLSConfig{
				Enabled: true,
				CertFiles: []string{
					cfg.Fabric.CA.TLSTrustedCerts,
				},
				Client: tls.KeyCertFiles{
					KeyFile:  cfg.Fabric.CA.TLSKey,
					CertFile: cfg.Fabric.CA.TLSCert,
				},
			},
		},
	}

	if err := caClient.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize CA client: %w", err)
	}

	return &CAClient{caClient: caClient}, nil
}

// Enroll performs the enrollment process for a user or administrator.
// It takes an api.EnrollmentRequest and returns the resulting lib.Identity or an error.
func (c *CAClient) Enroll(request *api.EnrollmentRequest) (*lib.Identity, error) {
	er, err := c.caClient.Enroll(request)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll: %w", err)
	}

	identity := er.Identity
	if identity == nil {
		return nil, fmt.Errorf("received empty identity after enrollment")
	}

	certFilePath := filepath.Join(identity.GetClient().Config.MSPDir, "signcerts", "cert.pem")
	if err := writeCert(identity, certFilePath); err != nil {
		return nil, fmt.Errorf("failed to save cert to file: %w", err)
	}

	return identity, nil
}

// EnrollAdmin enrolls the admin user and returns the admin identity.
// If an admin identity is already loaded, it is returned directly. Otherwise, it attempts to enroll
// the admin using credentials from environment variables or provided arguments.
func (c *CAClient) EnrollAdmin() (*lib.Identity, error) {
	identity, err := c.caClient.LoadMyIdentity()
	if err == nil {
		return identity, nil
	}

	return c.Enroll(&api.EnrollmentRequest{
		Name:    cfg.Fabric.CA.Admin,
		Secret:  cfg.Fabric.CA.AdminSecret,
		Profile: "tls",
		Type:    "x509",
	})
}

// RegisterAndEnrollUser registers and enrolls a new user using the provided admin identity.
// It creates a user directory, initializes CA clients for the admin and the new user,
// enrolls the admin, registers the new user, and then enrolls the new user.
func RegisterAndEnrollUser(regReq api.RegistrationRequest) (*lib.Identity, error) {
	userDir := filepath.Join(cfg.Fabric.CA.ClientHome, "users", regReq.Name)
	if err := createUserDir(userDir); err != nil {
		return nil, fmt.Errorf("failed to create user directory: %w", err)
	}

	userCAClient, err := NewCAClient(&cfg, userDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user CA client: %w", err)
	}

	adminCAClient, err := NewCAClient(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize admin CA client: %w", err)
	}

	adminIdentity, err := adminCAClient.EnrollAdmin()
	if err != nil {
		return nil, fmt.Errorf("failed to enroll admin: %w", err)
	}

	rr, err := adminIdentity.Register(&regReq)
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %v", err)
	}

	return userCAClient.Enroll(&api.EnrollmentRequest{
		Name:    regReq.Name,
		Secret:  rr.Secret,
		Profile: "tls",
		Type:    "x509",
	})
}

// certFromIdentity retrieves the certificate string from the given identity.
func certFromIdentity(identity *lib.Identity) (string, error) {
	credVal, err := identity.GetX509Credential().Val()
	if err != nil {
		return "", fmt.Errorf("failed to get x509 credential: %w", err)
	}

	signer, ok := credVal.(*x509.Signer)
	if !ok {
		return "", fmt.Errorf("failed to cast credential to x509.Signer")
	}

	return string(signer.Cert()), nil
}

// writeCert writes the certificate string extracted from the given identity to the specified file path.
func writeCert(identity *lib.Identity, filePath string) error {
	certString, err := certFromIdentity(identity)
	if err != nil {
		return fmt.Errorf("failed to get certificate from identity: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Close the file and check for an error
	defer func() {
		if cerr := file.Close(); cerr != nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	if _, err := file.WriteString(certString); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return err
}

// createUserDir creates a user directory with the necessary structure for storing MSP artifacts.
func createUserDir(userDir string) error {
	err := os.MkdirAll(filepath.Join(userDir, "msp"), 0700)
	if err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}

	return nil
}

// GetMSPKeyfile finds the first key file in the keystore directory within the specified home directory.
func GetMSPKeyfile(homeDir string) (string, error) {
	keystoreDir := filepath.Join(homeDir, "msp", "keystore")

	// Use filepath.Glob to directly find files matching the pattern
	keyFiles, err := filepath.Glob(filepath.Join(keystoreDir, "*"))
	if err != nil {
		return "", err
	}

	if len(keyFiles) == 0 {
		return "", fmt.Errorf("no key file found in keystore")
	}

	// Return the first key file found
	return keyFiles[0], nil
}

// LoadMSPKeyCert loads the key and certificate from the user's directory.
func LoadMSPKeyCert(homeDir string) (*MSPKeyCert, error) {
	certFilePath := filepath.Join(homeDir, "msp", "signcerts", "cert.pem")

	keyFilePath, err := GetMSPKeyfile(filepath.Join(homeDir))
	if err != nil {
		return nil, fmt.Errorf("failed to get MSP keyfile: %w", err)
	}

	certBytes, err := os.ReadFile(certFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyBytes, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	return &MSPKeyCert{
		Cert: string(certBytes),
		Key:  string(keyBytes),
	}, nil
}
