package ffmpg

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/disgoorg/ffmpeg-audio"
)

type FrameProvider interface {
	// ProvidePCMFrame is called to get a PCM frame.
	ProvidePCMFrame() ([]int16, error)

	// Close is called when the provider is no longer needed. It should close any open resources.
	Close()
}

const (
	// Exec is the default path to the ffmpeg executable
	Exec       = "ffmpeg"
	Channels   = 2
	SampleRate = 48000
	BufferSize = 65307
)

var _ FrameProvider = (*AudioProvider)(nil)

func New(ctx context.Context, opts ...ffmpeg.ConfigOpt) (*AudioProvider, error) {
	cfg := ffmpeg.DefaultConfig()
	cfg.Apply(opts)

	cmd := exec.CommandContext(ctx, cfg.Exec,
		"-thread_queue_size", "512",
		"-f", "alsa",
		"-channels", strconv.Itoa(cfg.Channels),
		"-i", "hw:2,0",
		"-ac", strconv.Itoa(cfg.Channels),
		"-ar", strconv.Itoa(cfg.SampleRate),
		"-af", "afftdn=nr=10,arnndn=m=/opt/golte/std.rnnn,lowpass=f=6000,highpass=f=150",
		"-f", "s16le",
		"-fflags", "+genpts+igndts",
		"-avoid_negative_ts", "make_zero",
		"-copyts",
		"pipe:1",
	)
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	done, doneFunc := context.WithCancel(context.Background())
	return &AudioProvider{
		cmd:      cmd,
		pipe:     pipe,
		reader:   bufio.NewReaderSize(pipe, cfg.BufferSize),
		channels: cfg.Channels,
		done:     done,
		doneFunc: doneFunc,
	}, nil
}

type AudioProvider struct {
	cmd      *exec.Cmd
	pipe     io.Closer
	reader   *bufio.Reader
	channels int
	done     context.Context
	doneFunc context.CancelFunc
}

func (p *AudioProvider) ProvidePCMFrame() ([]int16, error) {
	// Calculate frame size: 960 samples per frame * channels * 2 bytes per sample
	frameSize := 960 * p.channels * 2
	buf := make([]byte, frameSize)

	_, err := io.ReadFull(p.reader, buf)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
			p.doneFunc()
			return nil, io.EOF
		}
		return nil, fmt.Errorf("error reading PCM data: %w", err)
	}

	// Convert bytes to int16 samples
	samples := make([]int16, len(buf)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
	}

	return samples, nil
}

func (p *AudioProvider) Close() {
	_ = p.pipe.Close()
	p.doneFunc()
}

func (p *AudioProvider) Wait() error {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-p.done.Done()
	}()

	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = p.cmd.Wait()
	}()

	wg.Wait()
	return err
}
