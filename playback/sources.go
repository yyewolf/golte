package playback

import (
	"fmt"

	"golte/assets"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
)

// GetStreamer implements StreamSource for PredecodedSource
func (p *PredecodedSource) GetStreamer() (beep.Streamer, beep.Format, error) {
	cache := assets.GetPredecodedCache()
	audio, exists := cache.GetAudio(p.FilePath)
	if !exists {
		return nil, beep.Format{}, fmt.Errorf("predecoded audio not found: %s", p.FilePath)
	}

	// Create a new streamer from the buffer
	streamer := audio.Buffer.Streamer(3000, audio.Buffer.Len()-7000)
	return &effects.Volume{
		Streamer: streamer,
		Base:     2,
		Volume:   -0.5,
	}, audio.Format, nil
}
