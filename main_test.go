package main

import (
	"testing"
	"time"

	"golte/config"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				Discord: config.DiscordConfig{
					Token:      "test-token",
					WebhookURL: "https://discord.com/api/webhooks/test",
				},
				Modem: config.ModemConfig{
					Device:  "/dev/ttyUSB0",
					Baud:    115200,
					Timeout: 20 * time.Second,
				},
				Logging: config.LoggingConfig{
					Level:  "info",
					Format: "text",
				},
			},
			wantErr: false,
		},
		{
			name: "missing discord token",
			config: &config.Config{
				Discord: config.DiscordConfig{
					WebhookURL: "https://discord.com/api/webhooks/test",
				},
			},
			wantErr: true,
		},
		{
			name: "missing webhook URL",
			config: &config.Config{
				Discord: config.DiscordConfig{
					Token: "test-token",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
