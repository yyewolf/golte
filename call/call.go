package call

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/info"
)

// IncomingCallHandler is a callback for incoming calls with phone number
type IncomingCallHandler func(phoneNumber string)

// DTMFHandler is a callback for DTMF tone detection with the detected digit
type DTMFHandler func(digit string)

// Call represents a call manager that wraps AT modem functionality
type Call struct {
	*at.AT
	incomingHandler IncomingCallHandler
	dtmfHandler     DTMFHandler
	isListening     bool
	indicationMutex sync.RWMutex
}

// New creates a new Call manager with the provided AT modem
func New(a *at.AT) *Call {
	return &Call{
		AT: a,
	}
}

// CallStatus represents the status of a call
type CallStatus struct {
	Index     int    // Call index
	Direction string // Direction: "MO" (Mobile Originated) or "MT" (Mobile Terminated)
	Status    string // Status: "ACTIVE", "HELD", "DIALING", "ALERTING", "INCOMING", "WAITING"
	Mode      string // Mode: "VOICE", "DATA", "FAX"
	Number    string // Phone number
	Type      string // Number type
}

// StartCall initiates a call to the specified number
// Uses ATD command for voice calls
func (c *Call) StartCall(number string, options ...at.CommandOption) error {
	cmd := fmt.Sprintf("D%s;", number) // ATD with semicolon for voice call
	_, err := c.Command(cmd, options...)
	return err
}

// PickUp answers an incoming call
// Uses ATA command
func (c *Call) PickUp(options ...at.CommandOption) error {
	_, err := c.Command("A", options...)
	return err
}

// HangUp terminates the current call or all calls
// Uses ATH command
func (c *Call) HangUp(options ...at.CommandOption) error {
	_, err := c.Command("+CHUP", options...)
	return err
}

// HangUpSpecific hangs up a specific call by index
// Uses AT+CHUP or AT+CHLD=1x commands
func (c *Call) HangUpSpecific(callIndex int, options ...at.CommandOption) error {
	if callIndex <= 0 {
		// Hang up all calls
		return c.HangUp(options...)
	}

	// Try CHUP first (simpler command for hanging up)
	cmd := fmt.Sprintf("+CHLD=1%d", callIndex)
	_, err := c.Command(cmd, options...)
	return err
}

// GetCallStatus retrieves the status of all current calls
// Uses AT+CLCC command (List Current Calls)
func (c *Call) GetCallStatus(options ...at.CommandOption) ([]CallStatus, error) {
	response, err := c.Command("+CLCC", options...)
	if err != nil {
		return nil, err
	}

	var calls []CallStatus
	for _, line := range response {
		if info.HasPrefix(line, "+CLCC") {
			call, parseErr := parseCallStatus(line)
			if parseErr == nil {
				calls = append(calls, call)
			}
		}
	}

	return calls, nil
}

// parseCallStatus parses a +CLCC response line into a CallStatus struct
// Format: +CLCC: <id1>,<dir>,<stat>,<mode>,<mpty>[,<number>,<type>[,<alpha>[,<priority>]]]
func parseCallStatus(line string) (CallStatus, error) {
	// Remove the "+CLCC: " prefix
	data := info.TrimPrefix(line, "+CLCC")
	parts := strings.Split(data, ",")

	if len(parts) < 5 {
		return CallStatus{}, fmt.Errorf("invalid CLCC response format")
	}

	var call CallStatus
	var err error

	// Parse call index
	call.Index, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return CallStatus{}, fmt.Errorf("invalid call index: %w", err)
	}

	// Parse direction (0=MO, 1=MT)
	switch strings.TrimSpace(parts[1]) {
	case "0":
		call.Direction = "MO"
	case "1":
		call.Direction = "MT"
	default:
		call.Direction = "UNKNOWN"
	}

	// Parse status
	switch strings.TrimSpace(parts[2]) {
	case "0":
		call.Status = "ACTIVE"
	case "1":
		call.Status = "HELD"
	case "2":
		call.Status = "DIALING"
	case "3":
		call.Status = "ALERTING"
	case "4":
		call.Status = "INCOMING"
	case "5":
		call.Status = "WAITING"
	default:
		call.Status = "UNKNOWN"
	}

	// Parse mode
	switch strings.TrimSpace(parts[3]) {
	case "0":
		call.Mode = "VOICE"
	case "1":
		call.Mode = "DATA"
	case "2":
		call.Mode = "FAX"
	default:
		call.Mode = "UNKNOWN"
	}

	// Parse number if available
	if len(parts) > 5 {
		call.Number = strings.Trim(strings.TrimSpace(parts[5]), "\"")
	}

	// Parse number type if available
	if len(parts) > 6 {
		call.Type = strings.TrimSpace(parts[6])
	}

	return call, nil
}

// Mute enables or disables microphone muting
// Uses AT+CMUT command
func (c *Call) Mute(enable bool, options ...at.CommandOption) error {
	var cmd string
	if enable {
		cmd = "+CMUT=1" // Mute microphone
	} else {
		cmd = "+CMUT=0" // Unmute microphone
	}
	_, err := c.Command(cmd, options...)
	return err
}

// GetMuteStatus retrieves the current mute status
// Uses AT+CMUT? command
func (c *Call) GetMuteStatus(options ...at.CommandOption) (bool, error) {
	response, err := c.Command("+CMUT?", options...)
	if err != nil {
		return false, err
	}

	for _, line := range response {
		if info.HasPrefix(line, "+CMUT") {
			data := info.TrimPrefix(line, "+CMUT")
			status := strings.TrimSpace(data)
			return status == "1", nil
		}
	}

	return false, fmt.Errorf("no mute status found in response")
}

// VMute controls voice muting during calls
// Uses AT+VMUTE command
func (c *Call) VMute(enable bool, options ...at.CommandOption) error {
	var cmd string
	if enable {
		cmd = "+VMUTE=1" // Enable voice mute
	} else {
		cmd = "+VMUTE=0" // Disable voice mute
	}
	_, err := c.Command(cmd, options...)
	return err
}

// GetVMuteStatus retrieves the current voice mute status
// Uses AT+VMUTE? command
func (c *Call) GetVMuteStatus(options ...at.CommandOption) (bool, error) {
	response, err := c.Command("+VMUTE?", options...)
	if err != nil {
		return false, err
	}

	for _, line := range response {
		if info.HasPrefix(line, "+VMUTE") {
			data := info.TrimPrefix(line, "+VMUTE")
			status := strings.TrimSpace(data)
			return status == "1", nil
		}
	}

	return false, fmt.Errorf("no voice mute status found in response")
}

// StartListening begins listening for incoming calls using AT indications
// The handler will be called with the phone number when an incoming call is detected
func (c *Call) StartListening(handler IncomingCallHandler) error {
	c.indicationMutex.Lock()
	defer c.indicationMutex.Unlock()

	if c.isListening {
		return fmt.Errorf("already listening for incoming calls")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	c.incomingHandler = handler

	// Add indication for CLIP (Calling Line Identification Presentation)
	c.AddIndication("+CLIP", func(info []string) {
		if len(info) > 0 {
			phoneNumber := c.extractPhoneNumber(info[0])
			if phoneNumber != "" && c.incomingHandler != nil {
				c.incomingHandler(phoneNumber)
			}
		}
	})

	// Enable caller ID notifications
	_, err := c.Command("+CLIP=1")
	if err != nil {
		return fmt.Errorf("failed to enable caller ID: %w", err)
	}

	c.isListening = true
	return nil
}

// StopListening stops listening for incoming calls
func (c *Call) StopListening() error {
	c.indicationMutex.Lock()
	defer c.indicationMutex.Unlock()

	if !c.isListening {
		return fmt.Errorf("not currently listening")
	}

	// Remove the CLIP indication by setting it to nil
	c.AddIndication("+CLIP", nil)

	// Remove DTMF indication if it was set
	c.AddIndication("+RXDTMF", nil)

	// Disable caller ID notifications
	_, err := c.Command("+CLIP=0")
	if err != nil {
		log.Printf("Warning: failed to disable caller ID: %v", err)
	}

	// Disable DTMF detection (ignore errors as it might not be enabled)
	c.Command("+DDET=0")

	c.incomingHandler = nil
	c.dtmfHandler = nil
	c.isListening = false
	return nil
}

// IsListening returns whether the call manager is currently listening for incoming calls
func (c *Call) IsListening() bool {
	c.indicationMutex.RLock()
	defer c.indicationMutex.RUnlock()
	return c.isListening
}

// extractPhoneNumber extracts the phone number from a CLIP indication
// CLIP format: +CLIP: "number",type
func (c *Call) extractPhoneNumber(clipData string) string {
	// Use regex to extract phone number from CLIP indication
	reg := regexp.MustCompile(`\+CLIP:\s*"([^"]+)"`)
	matches := reg.FindStringSubmatch(clipData)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractDTMFDigit extracts the DTMF digit from a +RXDTMF indication
// RXDTMF format: +RXDTMF: digit
func (c *Call) extractDTMFDigit(dtmfData string) string {
	// Use regex to extract DTMF digit from RXDTMF indication
	reg := regexp.MustCompile(`\+RXDTMF:\s*([0-9A-D#*])`)
	matches := reg.FindStringSubmatch(dtmfData)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// SetDTMFHandler sets the DTMF detection handler
// The handler will be called with the detected DTMF digit (0-9, A-D, #, *)
func (c *Call) SetDTMFHandler(handler DTMFHandler) {
	c.indicationMutex.Lock()
	defer c.indicationMutex.Unlock()
	c.dtmfHandler = handler
}

// EnableDTMFDetection enables DTMF tone detection
// Uses AT+DDET command to enable DTMF detection
func (c *Call) EnableDTMFDetection(options ...at.CommandOption) error {
	c.indicationMutex.Lock()
	defer c.indicationMutex.Unlock()

	// Add indication for DTMF detection
	c.AddIndication("+RXDTMF", func(info []string) {
		if len(info) > 0 && c.dtmfHandler != nil {
			digit := c.extractDTMFDigit(info[0])
			if digit != "" {
				c.dtmfHandler(digit)
			}
		}
	})

	// Enable DTMF detection
	_, err := c.Command("+DDET=1", options...)
	if err != nil {
		return fmt.Errorf("failed to enable DTMF detection: %w", err)
	}

	return nil
}

// DisableDTMFDetection disables DTMF tone detection
func (c *Call) DisableDTMFDetection(options ...at.CommandOption) error {
	c.indicationMutex.Lock()
	defer c.indicationMutex.Unlock()

	// Remove DTMF indication
	c.AddIndication("+RXDTMF", nil)

	// Disable DTMF detection
	_, err := c.Command("+DDET=0", options...)
	if err != nil {
		log.Printf("Warning: failed to disable DTMF detection: %v", err)
	}

	c.dtmfHandler = nil
	return nil
}
