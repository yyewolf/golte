package machine

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"golte/call"
	"golte/config"
	"golte/playback"

	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/gsm"
	"github.com/warthog618/modem/serial"
)

// ModemManager handles all GSM modem operations
type ModemManager struct {
	config             *config.Config
	gsm                *gsm.GSM
	call               *call.Call
	playback           *playback.Playback
	logger             *slog.Logger
	callNotifyCallback func(from, message string)

	state *ModemState
}

type ModemState struct {
	password string
}

func NewState() *ModemState {
	return &ModemState{}
}

// NewModemManager creates a new ModemManager instance
func NewModemManager(cfg *config.Config, playback *playback.Playback, callNotifyCallback func(from, message string)) *ModemManager {
	return &ModemManager{
		config:             cfg,
		logger:             slog.With("component", "modem"),
		callNotifyCallback: callNotifyCallback,
		playback:           playback,
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

	at := at.New(mio,
		at.WithTimeout(m.config.Modem.Timeout),
		at.WithCmds("I"))

	m.gsm = gsm.New(at)
	m.call = call.New(at)

	if err := m.gsm.Init(); err != nil {
		serialModem.Close()
		return fmt.Errorf("failed to initialize modem: %w", err)
	}

	if err := m.call.Init(); err != nil {
		serialModem.Close()
		return fmt.Errorf("failed to initialize call manager: %w", err)
	}

	m.call.StartListening(func(call string) {
		message := fmt.Sprintf("📞 Incoming voice call")
		m.callNotifyCallback(call, message)
		m.call.PickUp()
		m.state = NewState()

		time.Sleep(1 * time.Second) // Wait for call to connect
		m.playback.AddPredecoded("audio/bonjour_veuillez_entrez_votre_mot_de_passe.mp3")
	})

	m.call.SetDTMFHandler(func(digit string) {
		m.logger.Info("DTMF digit received", slog.String("digit", digit))

		if digit == "#" {
			m.state = NewState()
			return
		}

		m.playback.AddPredecoded("audio/" + digit + ".mp3")
		m.state.password += digit

		if m.state.password == "52226636" {
			m.logger.Info("Password entered correctly")
			m.playback.AddPredecoded("audio/mot_de_passe_correct.mp3")
		}
	})
	m.call.EnableDTMFDetection()

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

// StartCall initiates a call to the specified number
func (m *ModemManager) StartCall(number string) error {
	m.logger.Info("Starting call",
		slog.String("number", number))

	err := m.call.StartCall(number)
	if err != nil {
		m.logger.Error("Failed to start call",
			slog.String("number", number),
			slog.Any("error", err))
		return err
	}

	m.logger.Info("Call initiated successfully", slog.String("number", number))
	return nil
}

// HangUpCall hangs up the current call
func (m *ModemManager) HangUpCall() error {
	m.logger.Info("Hanging up call")

	err := m.call.HangUp()
	if err != nil {
		m.logger.Error("Failed to hang up call",
			slog.Any("error", err))
		return err
	}

	m.logger.Info("Call hung up successfully")
	return nil
}

// Closed returns a channel that's closed when the modem connection is lost
func (m *ModemManager) Closed() <-chan struct{} {
	if m.gsm != nil {
		return m.gsm.Closed()
	}
	if m.call != nil {
		return m.call.Closed()
	}
	return nil
}

// GSM returns the underlying GSM instance
func (m *ModemManager) GSM() *gsm.GSM {
	return m.gsm
}
