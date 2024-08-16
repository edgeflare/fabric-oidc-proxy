package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:     "fabric-oidc-proxy",
	Short:   "A proxy service for Hyperledger Fabric using OIDC authentication",
	Version: "0.0.1",
}

func init() {
	// Persistent / Global flags
	rootCmd.PersistentFlags().String("config", "", "Config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().String("loglevel", "info", "Log level (debug, info, warn, error, dpanic, panic, fatal)")

	// bind persistent flags
	if err := viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config")); err != nil {
		fmt.Println("Error binding config flag:", err)
		os.Exit(1) // Or handle the error more gracefully
	}

	if err := viper.BindPFlag("loglevel", rootCmd.PersistentFlags().Lookup("loglevel")); err != nil {
		fmt.Println("Error binding loglevel flag:", err)
		os.Exit(1)
	}
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
