package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config represents the configuration for fabric-oidc-proxy
type Config struct {
	OIDC     OIDCConfig   `mapstructure:"oidc"`
	LogLevel string       `mapstructure:"loglevel"`
	HTTP     HTTPConfig   `mapstructure:"http"`
	Fabric   FabricConfig `mapstructure:"fabric"`
}

// OIDCConfig represents the configuration for OIDC
type OIDCConfig struct {
	Issuer       string `mapstructure:"issuer"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// HTTPConfig represents the configuration for the HTTP server
type HTTPConfig struct {
	Port int `mapstructure:"port"`
	TLS  struct {
		Cert string `mapstructure:"cert"`
		Key  string `mapstructure:"key"`
	} `mapstructure:"tls"`
}

// FabricConfig represents the configuration for the Fabric client
type FabricConfig struct {
	CA FabricCAConfig `mapstructure:"ca"`
	GW FabricGWConfig `mapstructure:"gw"`
}

// FabricConfig represents the configuration for the Fabric CA client
type FabricCAConfig struct {
	URL             string `mapstructure:"url"`
	ClientHome      string `mapstructure:"client_home"`
	ClientMSPDir    string `mapstructure:"client_mspdir"`
	TLSCert         string `mapstructure:"tls_cert"`
	TLSKey          string `mapstructure:"tls_key"`
	TLSTrustedCerts string `mapstructure:"tls_trusted_certs"`
	Admin           string `mapstructure:"admin"`
	AdminSecret     string `mapstructure:"admin_secret"`
	OIDCClaimKey    string `mapstructure:"oidc_claim_key"`
}

// FabricGWConfig represents the configuration for the Fabric Gateway client
type FabricGWConfig struct {
	MSPID                  string `mapstructure:"msp_id"`
	TLSCert                string `mapstructure:"tls_cert"`
	TLSKey                 string `mapstructure:"tls_key"`
	TLSTrustedCerts        string `mapstructure:"tls_trusted_certs"`
	PeerEndpoint           string `mapstructure:"peer_endpoint"`
	PeerServerNameOverride string `mapstructure:"peer_server_name_override"`
	MSPCert                string `mapstructure:"msp_cert"`
	MSPKey                 string `mapstructure:"msp_key"`
}

// LoadConfig loads the configuration from, in order of priority:
// 1. Command-line flags
// 2. Environment variables
// 3. Configuration file (config.yaml)
// 4. Default values
func LoadConfig(cmd *cobra.Command) (*Config, *zap.Logger, error) {
	// Bind command-line flags to Viper (done in Cobra commands setup)

	// Set up Viper to read environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// ... other defaults

	// Set configuration file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".") // Look for config.yaml in the current directory

	// Read configuration file (if it exists)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, nil, err // Handle other errors (e.g., parsing errors)
		}
		// If the file doesn't exist, it's okay, use defaults without config file
	}

	// Set defaults (use Fabric's defaults where applicable)
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("loglevel", "info")
	viper.SetDefault("fabric.ca.url", "http://localhost:7054")
	viper.SetDefault("fabric.ca.admin", "admin")
	viper.SetDefault("fabric.ca.admin_secret", "adminpw")
	viper.SetDefault("fabric.ca.client_mspdir", "msp")
	viper.SetDefault("fabric.ca.oidc_claim_key", "fabric")
	// Check if fabric/tls exists, create if not
	wd, _ := os.Getwd()
	tlsDirPath := filepath.Join(wd, "fabric", "tls")
	if _, err := os.Stat(tlsDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(tlsDirPath, 0755); err != nil { // Create with appropriate permissions
			return nil, nil, fmt.Errorf("failed to create fabric/tls directory: %w", err)
		}
	}
	viper.SetDefault("fabric.ca.client_home", "fabric")
	viper.SetDefault("fabric.ca.tls_trusted_certs", filepath.Join(tlsDirPath, "ca.crt"))

	viper.SetDefault("fabric.gw.msp_id", "Org1MSP")
	viper.SetDefault("fabric.gw.peer_endpoint", "dns:///127.0.0.1:7051")
	viper.SetDefault("fabric.gw.peer_server_name_override", "peer0.org1")
	viper.SetDefault("fabric.gw.tls_trusted_certs", filepath.Join(tlsDirPath, "ca.crt"))

	// bind config keys to environment variables
	viper.BindEnv("oidc.issuer")
	viper.BindEnv("oidc.client_id")
	viper.BindEnv("oidc.client_secret")

	viper.BindEnv("loglevel")

	viper.BindEnv("http.port")
	viper.BindEnv("http.tls.cert")
	viper.BindEnv("http.tls.key")

	viper.BindEnv("fabric.ca.url")
	viper.BindEnv("fabric.ca.client_home")
	viper.BindEnv("fabric.ca.client_mspdir")
	viper.BindEnv("fabric.ca.admin")
	viper.BindEnv("fabric.ca.admin_secret")
	viper.BindEnv("fabric.ca.client_tls_cert")
	viper.BindEnv("fabric.ca.client_tls_key")
	viper.BindEnv("fabric.ca.tls_trusted_certs")

	viper.BindEnv("fabric.gw.msp_id")
	viper.BindEnv("fabric.gw.tls_trusted_certs")
	viper.BindEnv("fabric.gw.peer_endpoint")
	viper.BindEnv("fabric.gw.peer_server_name_override")
	viper.BindEnv("fabric.gw.msp_cert")
	viper.BindEnv("fabric.gw.msp_key")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, nil, err
	}

	// Setup Zap logger
	logger, err := cfg.initLogger()
	if err != nil {
		return nil, nil, err
	}

	return &cfg, logger, nil
}

// initLogger initializes a zap logger based on the LogLevel in the configuration.
func (c *Config) initLogger() (*zap.Logger, error) {
	logLevel, err := zapcore.ParseLevel(c.LogLevel)
	if err != nil {
		return nil, err
	}

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(logLevel)
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
