package playback

import (
	"fmt"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// NewPlayback creates a new Playback instance
func NewPlayback(sampleRate beep.SampleRate) (*Playback, error) {
	// Initialize the speaker with the given sample rate
	err := speaker.Init(sampleRate, sampleRate.N(100*time.Millisecond))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize speaker: %w", err)
	}

	mixer := &beep.Mixer{}
	ctrl := &beep.Ctrl{Streamer: mixer}

	playback := &Playback{
		mixer:      mixer,
		ctrl:       ctrl,
		sampleRate: sampleRate,
		queue:      &Queue{},
	}

	mixer.Add(playback.queue)

	// Start playing the mixer
	speaker.Play(ctrl)

	return playback, nil
}

// AddStream adds a new audio stream to the playback mixer
func (p *Playback) AddStream(source StreamSource) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("playback is closed")
	}

	streamer, format, err := source.GetStreamer()
	if err != nil {
		return fmt.Errorf("failed to get streamer: %w", err)
	}

	// Resample if necessary to match the speaker's sample rate
	if format.SampleRate != p.sampleRate {
		resampled := beep.Resample(4, format.SampleRate, p.sampleRate, streamer)
		// Add to mixer
		p.mixer.Add(resampled)
		p.streamers = append(p.streamers, resampled)
	} else {
		// Add to mixer
		p.mixer.Add(streamer)
		p.streamers = append(p.streamers, streamer)
	}

	return nil
}

// AddPredecoded is a convenience method to add a predecoded audio file
func (p *Playback) AddPredecoded(filePath string) error {
	src := &PredecodedSource{FilePath: filePath}
	streamer, format, err := src.GetStreamer()
	if err != nil {
		return fmt.Errorf("failed to get streamer: %w", err)
	}

	resampled := beep.Resample(4, format.SampleRate, p.sampleRate, streamer)

	p.queue.Add(resampled)
	return nil
}

// SetVolume sets the volume for the entire playback (0.0 to 1.0)
func (p *Playback) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		speaker.Lock()
		p.mixer.Clear()
		// Re-add all streamers with volume control
		for _, s := range p.streamers {
			volumeStreamer := &effects.Volume{
				Streamer: s,
				Base:     2,
				Volume:   volume,
				Silent:   volume == 0,
			}
			p.mixer.Add(volumeStreamer)
		}
		speaker.Unlock()
	}
}

// Pause pauses the playback
func (p *Playback) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		speaker.Lock()
		p.ctrl.Paused = true
		speaker.Unlock()
	}
}

// Resume resumes the playback
func (p *Playback) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		speaker.Lock()
		p.ctrl.Paused = false
		speaker.Unlock()
	}
}

// Stop stops and clears all streams
func (p *Playback) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		speaker.Lock()
		p.mixer.Clear()
		p.streamers = nil
		speaker.Unlock()
	}
}

// Close closes the playback and releases resources
func (p *Playback) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	speaker.Lock()
	p.mixer.Clear()
	p.streamers = nil
	speaker.Unlock()

	// Close the speaker
	speaker.Close()

	return nil
}

// IsPlaying returns true if the playback is currently playing
func (p *Playback) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false
	}

	speaker.Lock()
	playing := !p.ctrl.Paused
	speaker.Unlock()

	return playing
}
