package playback

import (
	"errors"
	"time"

	"github.com/disgoorg/audio/pcm"
	"github.com/gopxl/beep/v2"
)

var ErrAlreadyClosed = errors.New("already closed")
var SampleRate = beep.SampleRate(48000)

type PCMStreamer struct {
	silence   *pcm.Packet
	pcm       []int16
	pcmIdx    int
	lastFrame [2]float64
	fadeLevel float64

	packets  chan *pcm.Packet
	closedCh chan struct{}
	closed   bool
}

var _ beep.Streamer = (*PCMStreamer)(nil)
var _ StreamSource = (*PCMStreamer)(nil)

func NewPCMStreamer(packets chan *pcm.Packet) *PCMStreamer {
	return &PCMStreamer{
		silence:  &pcm.Packet{PCM: make([]int16, SampleRate.N(20*time.Millisecond)*2)},
		packets:  packets,
		closedCh: make(chan struct{}),
	}
}

func (s *PCMStreamer) Err() error {
	return nil
}

func (s *PCMStreamer) Close() error {
	if s.closed {
		return ErrAlreadyClosed
	}
	s.closed = true
	go func() {
		s.closedCh <- struct{}{}
		close(s.closedCh)
	}()
	return nil
}

func (s *PCMStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	if s.closed {
		return 0, false
	}

	// State for concealment
	if s.fadeLevel == 0 {
		s.fadeLevel = 1.0 // start full volume
	}

	for n < len(samples) {
		if s.pcmIdx >= len(s.pcm) {
			select {
			case packet := <-s.packets:
				s.pcm = packet.PCM
				s.pcmIdx = 0
				s.fadeLevel = 1.0 // reset fade on new data
			case <-s.closedCh:
				return 0, false
			}
		}

		for ; n < len(samples) && s.pcmIdx < len(s.pcm); n++ {
			left := float64(s.pcm[s.pcmIdx]) / 32767
			right := float64(s.pcm[s.pcmIdx+1]) / 32767
			samples[n][0] = left
			samples[n][1] = right
			s.lastFrame = samples[n] // store for concealment
			s.pcmIdx += 2
		}
	}

	return n, true
}

// GetStreamer implements StreamSource for PCMStreamer
func (m *PCMStreamer) GetStreamer() (beep.Streamer, beep.Format, error) {
	return m, beep.Format{SampleRate: SampleRate, NumChannels: 2}, nil
}

// Reopen copies the streamer's state but resets the PCM buffer
func (s *PCMStreamer) Reopen() *PCMStreamer {
	return &PCMStreamer{
		silence:  s.silence,
		packets:  s.packets,
		closedCh: make(chan struct{}),
	}
}
