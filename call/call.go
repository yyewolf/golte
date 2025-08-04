package call

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/info"
)

// CallCallback is a function type for handling incoming calls
type CallCallback func(call CallStatus)

// Call represents a call manager that wraps AT modem functionality
type Call struct {
	*at.AT
	callback        CallCallback
	workerCtx       context.Context
	workerCancel    context.CancelFunc
	isWorkerRunning bool
}

// New creates a new Call manager with the provided AT modem and callback function
func New(a *at.AT, callback CallCallback) *Call {
	return &Call{
		AT:       a,
		callback: callback,
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

// StartWorker starts a background worker that monitors for incoming calls
// The worker polls the call status periodically and triggers the callback for new incoming calls
func (c *Call) StartWorker(pollInterval time.Duration) error {
	if c.isWorkerRunning {
		return fmt.Errorf("worker is already running")
	}

	if c.callback == nil {
		return fmt.Errorf("no callback function provided")
	}

	c.workerCtx, c.workerCancel = context.WithCancel(context.Background())
	c.isWorkerRunning = true

	go c.worker(pollInterval)
	return nil
}

// StopWorker stops the background worker
func (c *Call) StopWorker() {
	if c.isWorkerRunning && c.workerCancel != nil {
		c.workerCancel()
		c.isWorkerRunning = false
	}
}

// IsWorkerRunning returns whether the worker is currently running
func (c *Call) IsWorkerRunning() bool {
	return c.isWorkerRunning
}

// worker is the background goroutine that monitors for incoming calls
func (c *Call) worker(pollInterval time.Duration) {
	defer func() {
		c.isWorkerRunning = false
	}()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastKnownCalls map[int]CallStatus

	for {
		select {
		case <-c.workerCtx.Done():
			return
		case <-ticker.C:
			calls, err := c.GetCallStatus()
			if err != nil {
				log.Printf("Error retrieving call status: %v", err)
				// Log error but continue monitoring
				continue
			}

			currentCalls := make(map[int]CallStatus)
			for _, call := range calls {
				currentCalls[call.Index] = call
			}

			// Check for new incoming calls
			for index, call := range currentCalls {
				if lastCall, exists := lastKnownCalls[index]; !exists {
					// New call detected
					if call.Status == "INCOMING" && c.callback != nil {
						c.callback(call)
					}
				} else if lastCall.Status != call.Status && call.Status == "INCOMING" && c.callback != nil {
					// Call status changed to incoming
					c.callback(call)
				}
			}

			lastKnownCalls = currentCalls
		}
	}
}
