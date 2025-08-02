package cmd

import (
	"fmt"
	"log/slog"

	"golte/config"
	"golte/logger"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long:  "Commands for managing and validating golte configuration.",
}

// configValidateCmd validates the current configuration
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long:  "Validate the current configuration file and environment variables.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup basic logging for validation
		if err := logger.Setup("info", "text"); err != nil {
			return fmt.Errorf("failed to setup logging: %w", err)
		}

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate configuration
		if err := cfg.Validate(); err != nil {
			slog.Error("Configuration validation failed", slog.Any("error", err))
			return err
		}

		slog.Info("Configuration is valid")
		fmt.Println("✅ Configuration is valid")
		return nil
	},
}

// configShowCmd shows the current configuration
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display the current configuration values from file and environment variables.",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup basic logging
		if err := logger.Setup("info", "text"); err != nil {
			return fmt.Errorf("failed to setup logging: %w", err)
		}

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		fmt.Println("Current Configuration:")
		fmt.Printf("  Modem:\n")
		fmt.Printf("    Device: %s\n", cfg.Modem.Device)
		fmt.Printf("    Baud: %d\n", cfg.Modem.Baud)
		fmt.Printf("    Timeout: %s\n", cfg.Modem.Timeout)
		fmt.Printf("  Discord:\n")
		fmt.Printf("    Token: %s\n", maskToken(cfg.Discord.Token))
		fmt.Printf("    Webhook URL: %s\n", maskURL(cfg.Discord.WebhookURL))
		fmt.Printf("  Logging:\n")
		fmt.Printf("    Level: %s\n", cfg.Logging.Level)
		fmt.Printf("    Format: %s\n", cfg.Logging.Format)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
}

// maskToken masks a Discord token for display
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:8] + "***"
}

// maskURL masks a webhook URL for display
func maskURL(url string) string {
	if len(url) <= 20 {
		return "***"
	}
	return url[:20] + "***"
}
