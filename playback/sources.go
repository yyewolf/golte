package playback

import (
	"fmt"

	"golte/assets"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
)

// GetStreamer implements StreamSource for MP3Source
func (m *MP3Source) GetStreamer() (beep.Streamer, beep.Format, error) {
	file, err := m.FS.Open(m.FilePath)
	if err != nil {
		return nil, beep.Format{}, fmt.Errorf("failed to open MP3 file: %w", err)
	}

	streamer, format, err := mp3.Decode(file)
	if err != nil {
		file.Close()
		return nil, beep.Format{}, fmt.Errorf("failed to decode MP3: %w", err)
	}

	return streamer, format, nil
}

// GetStreamer implements StreamSource for PredecodedSource
func (p *PredecodedSource) GetStreamer() (beep.Streamer, beep.Format, error) {
	cache := assets.GetPredecodedCache()
	audio, exists := cache.GetAudio(p.FilePath)
	if !exists {
		return nil, beep.Format{}, fmt.Errorf("predecoded audio not found: %s", p.FilePath)
	}

	// Create a new streamer from the buffer
	streamer := audio.Buffer.Streamer(4096, audio.Buffer.Len()-8000)
	return streamer, audio.Format, nil
}
