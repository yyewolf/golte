package playback

import (
	"embed"
	"sync"

	"github.com/gopxl/beep/v2"
)

// Playback represents a single playback instance that can mix multiple audio streams
type Playback struct {
	mixer      *beep.Mixer
	ctrl       *beep.Ctrl
	mu         sync.RWMutex
	streamers  []beep.Streamer
	closed     bool
	sampleRate beep.SampleRate
	queue      *Queue
}

// StreamSource represents different types of audio input sources
type StreamSource interface {
	GetStreamer() (beep.Streamer, beep.Format, error)
}

// MP3Source represents an MP3 file source
type MP3Source struct {
	FS       embed.FS
	FilePath string
}

// PredecodedSource represents a predecoded audio source
type PredecodedSource struct {
	FilePath string
}
