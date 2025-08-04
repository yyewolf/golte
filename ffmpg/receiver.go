package ffmpg

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/snowflake/v2"
)

type FrameReceiver interface {
	// ReceivePCMFrame is called when a PCM frame is received.
	ReceivePCMFrame(userID snowflake.ID, packet *pcm.Packet) error

	// CleanupUser is called when a user is disconnected. This should close any resources associated with the user.
	CleanupUser(userID snowflake.ID)

	// Close is called when the receiver is no longer needed. It should close any open resources.
	Close()
}

type ALSAReceiver struct {
	mu     sync.RWMutex
	users  map[snowflake.ID]bool // Track active users
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewALSAReceiver() (*ALSAReceiver, error) {
	ctx, cancel := context.WithCancel(context.Background())

	receiver := &ALSAReceiver{
		users:  make(map[snowflake.ID]bool),
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Create the single FFmpeg instance
	if err := receiver.createFFmpegProcess(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create FFmpeg process: %w", err)
	}

	return receiver, nil
}

func (r *ALSAReceiver) createFFmpegProcess() error {
	r.cmd = exec.CommandContext(r.ctx, Exec,
		"-f", "s16le", // Raw PCM input format (16-bit little-endian)
		"-ar", strconv.Itoa(SampleRate), // Input sample rate (e.g., 48000)
		"-ac", strconv.Itoa(Channels), // Input channels (e.g., 2)
		"-i", "pipe:0", // Read from stdin
		"-f", "alsa", // Output format is ALSA
		"-ar", strconv.Itoa(SampleRate), // Output sample rate
		"-ac", strconv.Itoa(Channels), // Output channels
		"hw:2,0", // ALSA device
	)

	stdin, err := r.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	r.stdin = stdin

	r.cmd.Stderr = os.Stdout // Set stderr to nil to avoid capturing it

	// Start the command
	if err := r.cmd.Start(); err != nil {
		r.stdin.Close()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Monitor the process in a goroutine
	go func() {
		defer r.cancel()
		if err := r.cmd.Wait(); err != nil {
			fmt.Printf("ffmpeg process exited with error: %v\n", err)
			// print stderr if needed
			fmt.Printf("ffmpeg stderr: %s\n", r.cmd.Stderr) // Assuming stderr is set
			// Process ended unexpectedly
			r.mu.Lock()
			if r.stdin != nil {
				r.stdin.Close()
				r.stdin = nil
			}
			r.mu.Unlock()
		}
	}()

	return nil
}

func (r *ALSAReceiver) ReceivePCMFrame(userID snowflake.ID, packet *pcm.Packet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Track this user as active
	r.users[userID] = true

	// Write PCM data directly to ffmpeg stdin
	if r.stdin == nil {
		return fmt.Errorf("ffmpeg stdin is not available")
	}

	// Convert PCM data from []int16 to []byte (little-endian)
	buf := make([]byte, len(packet.PCM)*2) // 2 bytes per int16
	for i, sample := range packet.PCM {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(sample))
	}

	// Write the PCM audio data to ffmpeg stdin
	if _, err := r.stdin.Write(buf); err != nil {
		return fmt.Errorf("failed to write to ffmpeg stdin: %w", err)
	}

	return nil
}

func (r *ALSAReceiver) CleanupUser(userID snowflake.ID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, userID)
}

func (r *ALSAReceiver) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear all users
	r.users = make(map[snowflake.ID]bool)

	// Close stdin to signal FFmpeg to stop
	if r.stdin != nil {
		r.stdin.Close()
		r.stdin = nil
	}

	// Cancel the main context
	r.cancel()

	// Signal that we're done
	close(r.done)
}

func (r *ALSAReceiver) Wait() {
	<-r.done
}
