package ffmpeg

import (
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

// Player plays MP3 audio files from an embedded filesystem through FFmpeg
type Player struct {
	mu     sync.RWMutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// NewPlayer creates a new MP3 player that outputs to pipe:0 via FFmpeg
func NewPlayer() (*Player, error) {
	ctx, cancel := context.WithCancel(context.Background())

	player := &Player{
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	return player, nil
}

// PlayMP3 plays an MP3 file from the embedded filesystem
func (p *Player) PlayMP3(assets embed.FS, filename string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we're already playing something
	if p.cmd != nil {
		return fmt.Errorf("player is already active")
	}

	// Read the MP3 file from embedded filesystem
	data, err := assets.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read MP3 file %s: %w", filename, err)
	}

	// Create FFmpeg command to decode MP3 and output to pipe:0
	p.cmd = exec.CommandContext(p.ctx, Exec,
		"-f", "mp3", // Input format is MP3
		"-i", "pipe:0", // Read MP3 data from stdin
		"-f", "alsa", // Output format: ALSA
		"-ar", strconv.Itoa(SampleRate), // Output sample rate
		"-ac", strconv.Itoa(Channels), // Output channels
		"hw:2,0", // Output device (ALSA hardware device)
	)

	// Get stdin pipe to send MP3 data
	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		p.cmd = nil
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	p.stdin = stdin

	// Set stderr to capture any FFmpeg errors
	p.cmd.Stderr = os.Stderr

	// Start the FFmpeg process
	if err := p.cmd.Start(); err != nil {
		p.stdin.Close()
		p.stdin = nil
		p.cmd = nil
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Send MP3 data to FFmpeg in a goroutine
	go func() {
		defer p.stdin.Close()

		if _, err := p.stdin.Write(data); err != nil {
			fmt.Printf("Error writing MP3 data to ffmpeg: %v\n", err)
		}
	}()

	// Monitor the process completion
	go func() {
		defer func() {
			p.mu.Lock()
			p.cmd = nil
			p.stdin = nil
			p.mu.Unlock()
			close(p.done)
		}()

		if err := p.cmd.Wait(); err != nil {
			fmt.Printf("ffmpeg process exited with error: %v\n", err)
		}
	}()

	return nil
}

// PlayMP3Sync plays an MP3 file synchronously and waits for completion
func (p *Player) PlayMP3Sync(assets embed.FS, filename string) error {
	if err := p.PlayMP3(assets, filename); err != nil {
		return err
	}

	// Wait for playback to complete
	p.Wait()
	return nil
}

// IsPlaying returns true if the player is currently playing
func (p *Player) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cmd != nil
}

// Stop stops the current playback
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil {
		// Close stdin to signal FFmpeg to stop
		if p.stdin != nil {
			p.stdin.Close()
			p.stdin = nil
		}

		// Kill the process if it's still running
		if p.cmd.Process != nil {
			p.cmd.Process.Kill()
		}

		p.cmd = nil
	}

	// Cancel the context
	p.cancel()
}

// Wait blocks until playback is complete
func (p *Player) Wait() {
	<-p.done
}

// Close cleans up the player resources
func (p *Player) Close() {
	p.Stop()
}
