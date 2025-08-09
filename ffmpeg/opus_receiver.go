package ffmpeg

import (
	"golte/playback"

	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/snowflake/v2"
)

type OpusFrameReceiver interface {
	// ReceivePCMFrame is called when a PCM frame is received.
	ReceivePCMFrame(userID snowflake.ID, packet *pcm.Packet) error

	// CleanupUser is called when a user is disconnected. This should close any resources associated with the user.
	CleanupUser(userID snowflake.ID)

	// Close is called when the receiver is no longer needed. It should close any open resources.
	Close()
}

type OpusPCMReceiver struct {
	Ch chan *pcm.Packet
}

func NewOpusPCMReceiver() (*OpusPCMReceiver, *playback.PCMStreamer, error) {
	receiver := &OpusPCMReceiver{
		Ch: make(chan *pcm.Packet, 10),
	}

	streamer := playback.NewPCMStreamer(receiver.Ch)
	return receiver, streamer, nil
}

func (r *OpusPCMReceiver) ReceivePCMFrame(userID snowflake.ID, packet *pcm.Packet) error {
	r.Ch <- packet
	return nil
}

func (r *OpusPCMReceiver) CleanupUser(userID snowflake.ID) {
	// Cleanup any resources for the user
}

func (r *OpusPCMReceiver) Close() {
	close(r.Ch)
}
