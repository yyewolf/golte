package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golte/config"
	"golte/logger"
	"golte/machine"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "golte",
	Short: "A GSM/LTE modem to Discord bridge",
	Long: `Golte is a bridge application that connects a GSM/LTE modem to Discord,
allowing you to send and receive SMS messages through Discord commands and webhooks.

The application monitors incoming SMS messages and forwards them to a Discord webhook,
while also providing Discord slash commands to send SMS messages through the modem.`,
	RunE: runServer,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Local flags for the server command
	rootCmd.Flags().StringP("device", "d", "/dev/serial0", "path to modem device")
	rootCmd.Flags().IntP("baud", "b", 115200, "baud rate")
	rootCmd.Flags().Duration("timeout", 20*time.Second, "command timeout period")
	rootCmd.Flags().String("discord-token", "", "Discord bot token")
	rootCmd.Flags().String("discord-webhook", "", "Discord webhook URL")
	rootCmd.Flags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.Flags().String("log-format", "text", "log format (text, json)")

	// Bind flags to viper
	viper.BindPFlag("modem.device", rootCmd.Flags().Lookup("device"))
	viper.BindPFlag("modem.baud", rootCmd.Flags().Lookup("baud"))
	viper.BindPFlag("modem.timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("discord.token", rootCmd.Flags().Lookup("discord-token"))
	viper.BindPFlag("discord.webhook_url", rootCmd.Flags().Lookup("discord-webhook"))
	viper.BindPFlag("logging.level", rootCmd.Flags().Lookup("log-level"))
	viper.BindPFlag("logging.format", rootCmd.Flags().Lookup("log-format"))
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	if verbose {
		viper.Set("logging.level", "debug")
	}
}

// runServer starts the main application
func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Setup logging
	if err := logger.Setup(cfg.Logging.Level, cfg.Logging.Format); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	// Create and initialize the machine
	m := machine.New(cfg)
	if err := m.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize machine: %w", err)
	}

	// Start the machine
	if err := m.Start(); err != nil {
		return fmt.Errorf("failed to start machine: %w", err)
	}

	// Setup graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case sig := <-signalChan:
		fmt.Printf("\nReceived %s, shutting down gracefully...\n", sig)
	case err := <-m.Error():
		fmt.Printf("Error occurred: %v\n", err)
	}

	// Graceful shutdown
	if err := m.Stop(); err != nil {
		return fmt.Errorf("failed to stop machine gracefully: %w", err)
	}

	return nil
}
