package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ryuyb/litchi/internal/infrastructure/config"
	"github.com/spf13/cobra"
	"go.uber.org/fx/fxevent"
)

var (
	// cfgFile is the config file path (from CLI flag)
	cfgFile string
	// envMode is the environment for config selection (from CLI flag)
	envMode string
	// loadedCfg is the loaded configuration (available to subcommands)
	loadedCfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "litchi",
	Short: "Litchi - Automated development agent system",
	Long: `Litchi is an automated development agent system that transforms
GitHub Issues into Pull Requests through a five-stage workflow:
Clarification → Design → TaskBreakdown → Execution → PullRequest`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Load config directly using CLI options
		opts := config.LoadOptions{
			ConfigPath: cfgFile,
			Env:        config.Environment(envMode),
		}
		var err error
		loadedCfg, err = config.NewConfigWithOptions(opts)
		return err
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Try to print error to stderr, but proceed with exit even if write fails
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file path (default is ./config/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&envMode, "env", "e", "", "environment for config file selection (dev, uat, prod)")
}

// getFxLogger returns the appropriate Fx event logger based on server mode.
// Priority: explicit mode > loaded config server.mode > default (debug)
func getFxLogger(serverMode string) fxevent.Logger {
	if isDebugMode(serverMode) {
		return &fxevent.ConsoleLogger{W: os.Stderr}
	}
	return fxevent.NopLogger
}

// isDebugMode checks if the server should run in debug mode.
// Priority: explicit mode > loaded config server.mode > LITCHI_SERVER_MODE env > default
func isDebugMode(mode string) bool {
	// CLI flag takes highest priority
	if mode != "" {
		m := strings.ToLower(mode)
		return m == "debug" || m == "dev" || m == "development"
	}

	// Use loaded config if available
	if loadedCfg != nil {
		m := strings.ToLower(loadedCfg.Server.Mode)
		return m == "" || m == "debug" || m == "dev" || m == "development"
	}

	// Fallback to environment variable
	if m := os.Getenv("LITCHI_SERVER_MODE"); m != "" {
		return strings.ToLower(m) == "debug" || strings.ToLower(m) == "dev" || strings.ToLower(m) == "development"
	}

	// Default to debug mode
	return true
}