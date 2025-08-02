package machine

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"golte/config"

	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/gsm"
	"github.com/warthog618/modem/serial"
)

// ModemManager handles all GSM modem operations
type ModemManager struct {
	config *config.Config
	gsm    *gsm.GSM
	logger *slog.Logger
}

// NewModemManager creates a new ModemManager instance
func NewModemManager(cfg *config.Config) *ModemManager {
	return &ModemManager{
		config: cfg,
		logger: slog.With("component", "modem"),
	}
}

// Initialize sets up the GSM modem connection
func (m *ModemManager) Initialize() error {
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

// SendSMS sends an SMS message through the modem
func (m *ModemManager) SendSMS(number, message string) error {
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

// StartMessageReception begins listening for incoming SMS messages
func (m *ModemManager) StartMessageReception(onMessage func(gsm.Message), onError func(error)) error {
	m.logger.Info("Starting SMS message reception")

	err := m.gsm.StartMessageRx(onMessage, onError)
	if err != nil {
		return fmt.Errorf("failed to start message reception: %w", err)
	}

	m.logger.Info("SMS message reception started")
	return nil
}

// StopMessageReception stops SMS message reception
func (m *ModemManager) StopMessageReception() {
	if m.gsm != nil {
		m.gsm.StopMessageRx()
	}
}

// GetSignalQuality retrieves the current signal quality
func (m *ModemManager) GetSignalQuality() (interface{}, error) {
	return m.gsm.Command("+CSQ")
}

// Closed returns a channel that's closed when the modem connection is lost
func (m *ModemManager) Closed() <-chan struct{} {
	if m.gsm != nil {
		return m.gsm.Closed()
	}
	return nil
}

// GSM returns the underlying GSM instance
func (m *ModemManager) GSM() *gsm.GSM {
	return m.gsm
}
