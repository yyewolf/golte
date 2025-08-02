package machine

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golte/config"
)

// WebhookManager handles Discord webhook operations
type WebhookManager struct {
	config *config.Config
	logger *slog.Logger
	client *http.Client
}

// NewWebhookManager creates a new WebhookManager instance
func NewWebhookManager(cfg *config.Config) *WebhookManager {
	return &WebhookManager{
		config: cfg,
		logger: slog.With("component", "webhook"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendMessage sends a message to Discord via webhook
func (w *WebhookManager) SendMessage(from, message string) error {
	data := fmt.Sprintf(`{"username":"%s","content": "%s"}`, from, message)
	req, err := http.NewRequest("POST", w.config.Discord.WebhookURL, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, body)
	}

	w.logger.Debug("Forwarded SMS to Discord",
		slog.String("from", from),
		slog.Int("status", resp.StatusCode))

	return nil
}
