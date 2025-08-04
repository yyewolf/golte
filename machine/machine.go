package machine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"golte/config"

	"github.com/warthog618/modem/gsm"
)

// Machine represents the main application state
type Machine struct {
	config        *config.Config
	modem         *ModemManager
	discord       *DiscordManager
	signalMonitor *SignalMonitor
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	stopChan      chan struct{}
	errorChan     chan error
}

// New creates a new Machine instance
func New(cfg *config.Config) *Machine {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Machine{
		config:    cfg,
		logger:    slog.With("component", "machine"),
		ctx:       ctx,
		cancel:    cancel,
		stopChan:  make(chan struct{}),
		errorChan: make(chan error, 10),
	}

	// Initialize components
	m.modem = NewModemManager(cfg, m.sendCallNotification)
	m.signalMonitor = NewSignalMonitor(cfg, m.modem, &m.wg)

	// Discord manager needs SMS function and notification function
	m.discord = NewDiscordManager(cfg, m.SendSMS, m.StartCall, m.HangUpCall, m.sendDiscordEmbed)

	return m
}

// Initialize sets up the machine components
func (m *Machine) Initialize() error {
	m.logger.Info("Initializing machine...")

	// Initialize modem
	if err := m.modem.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize modem: %w", err)
	}

	// Initialize Discord client
	if err := m.discord.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize Discord: %w", err)
	}

	m.logger.Info("Machine initialized successfully")
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
	m.signalMonitor.SetContext(m.ctx)
	m.signalMonitor.Start()

	// Start Discord gateway
	if err := m.discord.Start(m.ctx); err != nil {
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

	// Stop signal monitoring
	if m.signalMonitor != nil {
		m.signalMonitor.Stop()
	}

	// Close Discord connection
	if m.discord != nil {
		m.discord.Stop()
	}

	// Stop SMS reception
	if m.modem != nil {
		m.modem.StopMessageReception()
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
	case <-m.modem.Closed():
		return fmt.Errorf("modem connection closed")
	}
}

// SendSMS sends an SMS message through the modem
func (m *Machine) SendSMS(number, message string) error {
	return m.modem.SendSMS(number, message)
}

// StartCall initiates a call through the modem
func (m *Machine) StartCall(number string) error {
	return m.modem.StartCall(number)
}

// HangUpCall hangs up the current call
func (m *Machine) HangUpCall() error {
	return m.modem.HangUpCall()
}

// Error returns the error channel for monitoring errors
func (m *Machine) Error() <-chan error {
	return m.errorChan
}

// startMessageReception begins listening for incoming SMS messages
func (m *Machine) startMessageReception() error {
	m.logger.Info("Starting SMS message reception")

	return m.modem.StartMessageReception(
		func(msg gsm.Message) {
			m.logger.Info("Received SMS",
				slog.String("from", msg.Number),
				slog.String("message", msg.Message))

			if err := m.discord.SendEmbed(NotificationTypeSMS, msg.Number, msg.Message); err != nil {
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
}

// sendDiscordEmbed sends a formatted embed to Discord
func (m *Machine) sendDiscordEmbed(notificationType NotificationType, from, message string) {
	if err := m.discord.SendEmbed(notificationType, from, message); err != nil {
		m.logger.Error("Failed to send Discord embed",
			slog.String("type", string(notificationType)),
			slog.String("from", from),
			slog.Any("error", err))
	}
}

// sendCallNotification sends a call notification to Discord
func (m *Machine) sendCallNotification(from, message string) {
	if err := m.discord.SendEmbed(NotificationTypeCall, from, message); err != nil {
		m.logger.Error("Failed to send call notification to Discord",
			slog.String("from", from),
			slog.Any("error", err))
	}
}
