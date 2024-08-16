package cmd

import (
	"fmt"

	"github.com/edgeflare/fabric-oidc-proxy/internal/config"
	"github.com/edgeflare/fabric-oidc-proxy/internal/fabric"
	"github.com/edgeflare/fabric-oidc-proxy/internal/proxy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Fabric OIDC proxy server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, logger, err := config.LoadConfig(cmd)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		logger.Info("Configuration loaded successfully")

		// initialize fabric CA client and enroll admin
		fabric.Init(cfg, logger)
		adminCAClient, err := fabric.NewCAClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create CA client: %w", err)
		}

		_, err = adminCAClient.EnrollAdmin()
		if err != nil {
			return fmt.Errorf("failed to enroll admin: %w", err)
		}

		err = proxy.StartServer(cfg, logger) // Pass the logger here
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

func init() {
	// Server-specific flags
	startCmd.Flags().Int("port", 8080, "Server port (overrides config file)")
	if err := viper.BindPFlag("http.port", startCmd.Flags().Lookup("port")); err != nil {
		panic(fmt.Sprintf("Error binding port flag: %v", err))
	}

	startCmd.Flags().String("oidc.issuer", "", "OIDC Issuer URL")
	if err := viper.BindPFlag("oidc.issuer", startCmd.Flags().Lookup("oidc.issuer")); err != nil {
		panic(fmt.Sprintf("Error binding OIDC issuer flag: %v", err))
	}
	startCmd.Flags().String("oidc.client_id", "", "OIDC Client ID")
	if err := viper.BindPFlag("oidc.client_id", startCmd.Flags().Lookup("oidc.client_id")); err != nil {
		panic(fmt.Sprintf("Error binding OIDC client ID flag: %v", err))
	}
	startCmd.Flags().String("oidc.client_secret", "", "OIDC Client Secret")
	if err := viper.BindPFlag("oidc.client_secret", startCmd.Flags().Lookup("oidc.client_secret")); err != nil {
		panic(fmt.Sprintf("Error binding OIDC client secret flag: %v", err))
	}

	// TODO: other flags

	rootCmd.AddCommand(startCmd)
}
