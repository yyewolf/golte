package machine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"golte/config"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/gsm"
	"github.com/warthog618/modem/serial"
)

// Machine represents the main application state
type Machine struct {
	config    *config.Config
	gsm       *gsm.GSM
	discord   bot.Client
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	stopChan  chan struct{}
	errorChan chan error
}

// New creates a new Machine instance
func New(cfg *config.Config) *Machine {
	ctx, cancel := context.WithCancel(context.Background())

	return &Machine{
		config:    cfg,
		logger:    slog.With("component", "machine"),
		ctx:       ctx,
		cancel:    cancel,
		stopChan:  make(chan struct{}),
		errorChan: make(chan error, 10),
	}
}

// Initialize sets up the machine components
func (m *Machine) Initialize() error {
	m.logger.Info("Initializing machine...")

	// Initialize modem
	if err := m.initializeModem(); err != nil {
		return fmt.Errorf("failed to initialize modem: %w", err)
	}

	// Initialize Discord client
	if err := m.initializeDiscord(); err != nil {
		return fmt.Errorf("failed to initialize Discord: %w", err)
	}

	m.logger.Info("Machine initialized successfully")
	return nil
}

// initializeModem sets up the GSM modem connection
func (m *Machine) initializeModem() error {
	m.logger.Info("Initializing modem",
		slog.String("device", m.config.Modem.Device),
		slog.Int("baud", m.config.Modem.Baud))

	serialModem, err := serial.New(
		serial.WithPort(m.config.Modem.Device),
		serial.WithBaud(m.config.Modem.Baud),
	)
	if err != nil {
		return fmt.Errorf("failed to create serial connection: %w", err)
	}

	var mio io.ReadWriter = serialModem

	m.gsm = gsm.New(at.New(mio,
		at.WithTimeout(m.config.Modem.Timeout),
		at.WithCmds("I")))

	if err := m.gsm.Init(); err != nil {
		serialModem.Close()
		return fmt.Errorf("failed to initialize modem: %w", err)
	}

	m.logger.Info("Modem initialized successfully")
	return nil
}

// initializeDiscord sets up the Discord bot client
func (m *Machine) initializeDiscord() error {
	m.logger.Info("Initializing Discord client")

	client, err := disgo.New(m.config.Discord.Token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(m.commandListener),
	)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	m.discord = client

	// Register commands
	if _, err = client.Rest().SetGlobalCommands(client.ApplicationID(), m.getCommands()); err != nil {
		return fmt.Errorf("failed to register Discord commands: %w", err)
	}

	m.logger.Info("Discord client initialized successfully")
	return nil
}

// Start begins all machine operations
func (m *Machine) Start() error {
	m.logger.Info("Starting machine operations...")

	// Start message reception
	if err := m.startMessageReception(); err != nil {
		return fmt.Errorf("failed to start message reception: %w", err)
	}

	// Start signal quality polling
	m.startSignalQualityPolling()

	// Start Discord gateway
	if err := m.discord.OpenGateway(m.ctx); err != nil {
		return fmt.Errorf("failed to connect to Discord gateway: %w", err)
	}

	m.logger.Info("Machine started successfully")
	return nil
}

// Stop gracefully shuts down the machine
func (m *Machine) Stop() error {
	m.logger.Info("Stopping machine...")

	// Cancel context to stop all operations
	m.cancel()

	// Close Discord connection
	if m.discord != nil {
		m.discord.Close(context.Background())
	}

	// Stop SMS reception
	if m.gsm != nil {
		m.gsm.StopMessageRx()
	}

	// Wait for all goroutines to finish
	m.wg.Wait()

	m.logger.Info("Machine stopped")
	return nil
}

// Wait blocks until the machine is stopped
func (m *Machine) Wait() error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	case err := <-m.errorChan:
		return err
	case <-m.gsm.Closed():
		return fmt.Errorf("modem connection closed")
	}
}

// SendSMS sends an SMS message through the modem
func (m *Machine) SendSMS(number, message string) error {
	m.logger.Info("Sending SMS",
		slog.String("number", number),
		slog.Int("length", len(message)))

	var err error
	if len(message) > 160 {
		// Long SMS, split into multiple messages
		_, err = m.gsm.SendLongMessage(number, message, at.WithTimeout(5*time.Second))
	} else {
		_, err = m.gsm.SendShortMessage(number, message, at.WithTimeout(5*time.Second))
	}

	if err != nil {
		m.logger.Error("Failed to send SMS",
			slog.String("number", number),
			slog.Any("error", err))
		return err
	}

	m.logger.Info("SMS sent successfully", slog.String("number", number))
	return nil
}

// Error returns the error channel for monitoring errors
func (m *Machine) Error() <-chan error {
	return m.errorChan
}

// getCommands returns the Discord slash commands
func (m *Machine) getCommands() []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "send",
			Description: "sends a SMS",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "number",
					Description: "The phone number to send the message to",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "message",
					Description: "What to say",
					Required:    true,
				},
			},
		},
	}
}

// commandListener handles Discord slash commands
func (m *Machine) commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "send" {
		phoneNumber := data.String("number")
		message := data.String("message")

		m.logger.Info("Received SMS command from Discord",
			slog.String("number", phoneNumber),
			slog.String("user", event.User().Username))

		err := m.SendSMS(phoneNumber, message)
		if err != nil {
			m.logger.Error("Failed to send SMS via Discord command",
				slog.String("number", phoneNumber),
				slog.Any("error", err))

			err = event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("SMS has **not** been sent: %v", err).
				SetEphemeral(true).
				Build())
			if err != nil {
				m.logger.Error("Failed to send Discord response", slog.Any("error", err))
			}
			return
		}

		err = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("SMS Sent!").
			SetEphemeral(true).
			Build())
		if err != nil {
			m.logger.Error("Failed to send Discord response", slog.Any("error", err))
		}
	}
}

// startMessageReception begins listening for incoming SMS messages
func (m *Machine) startMessageReception() error {
	m.logger.Info("Starting SMS message reception")

	err := m.gsm.StartMessageRx(
		func(msg gsm.Message) {
			m.logger.Info("Received SMS",
				slog.String("from", msg.Number),
				slog.String("message", msg.Message))

			if err := m.sendDiscordMessage(msg.Number, msg.Message); err != nil {
				m.logger.Error("Failed to forward SMS to Discord",
					slog.String("from", msg.Number),
					slog.Any("error", err))
			}
		},
		func(err error) {
			m.logger.Error("SMS reception error", slog.Any("error", err))
			select {
			case m.errorChan <- err:
			default:
			}
		})

	if err != nil {
		return fmt.Errorf("failed to start message reception: %w", err)
	}

	m.logger.Info("SMS message reception started")
	return nil
}

// startSignalQualityPolling starts a goroutine to monitor signal quality
func (m *Machine) startSignalQualityPolling() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		logger := m.logger.With("component", "signal-monitor")
		logger.Info("Starting signal quality monitoring")

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				result, err := m.gsm.Command("+CSQ")
				if err != nil {
					logger.Error("Failed to get signal quality", slog.Any("error", err))
				} else {
					logger.Debug("Signal quality", slog.Any("result", result))
				}
			case <-m.ctx.Done():
				logger.Info("Signal quality monitoring stopped")
				return
			case <-m.gsm.Closed():
				logger.Warn("Modem closed, stopping signal quality monitoring")
				return
			}
		}
	}()
}

// sendDiscordMessage sends a message to Discord via webhook
func (m *Machine) sendDiscordMessage(from, message string) error {
	data := fmt.Sprintf(`{"username":"%s","content": "%s"}`, from, message)
	req, err := http.NewRequest("POST", m.config.Discord.WebhookURL, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned status %d: %s", resp.StatusCode, body)
	}

	m.logger.Debug("Forwarded SMS to Discord",
		slog.String("from", from),
		slog.Int("status", resp.StatusCode))

	return nil
}
