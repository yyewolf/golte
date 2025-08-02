package config

import (
	"log/slog"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Modem configuration
	Modem ModemConfig `mapstructure:"modem"`

	// Discord configuration
	Discord DiscordConfig `mapstructure:"discord"`

	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging"`
}

// ModemConfig holds modem-specific configuration
type ModemConfig struct {
	Device  string        `mapstructure:"device"`
	Baud    int           `mapstructure:"baud"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// DiscordConfig holds Discord-specific configuration
type DiscordConfig struct {
	Token      string `mapstructure:"token"`
	WebhookURL string `mapstructure:"webhook_url"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json or text
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	// Set defaults
	viper.SetDefault("modem.device", "/dev/serial0")
	viper.SetDefault("modem.baud", 115200)
	viper.SetDefault("modem.timeout", "20s")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")

	// Read config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.golte")
	viper.AddConfigPath("/etc/golte")

	// Allow environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("GOLTE")

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		slog.Debug("No config file found, using defaults and environment variables")
	} else {
		slog.Info("Using config file", slog.String("file", viper.ConfigFileUsed()))
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateConfig validates the configuration
func (c *Config) Validate() error {
	if c.Discord.Token == "" {
		return &ConfigError{Field: "discord.token", Message: "Discord token is required"}
	}
	if c.Discord.WebhookURL == "" {
		return &ConfigError{Field: "discord.webhook_url", Message: "Discord webhook URL is required"}
	}
	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}
