package machine

import (
	"context"
	"golte/ffmpeg"
	"log"
	"os"
	"syscall"

	"github.com/disgoorg/audio/opus"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/voice"
	disgoorgffmpeg "github.com/disgoorg/ffmpeg-audio"
	"github.com/disgoorg/snowflake/v2"
)

func (d *DiscordManager) ConnectAndPlay(closeChan chan os.Signal) {
	guild_id := snowflake.MustParse(d.config.Discord.GuildID)
	vc_id := snowflake.MustParse(d.config.Discord.VoiceChannelID)

	conn := d.client.VoiceManager().CreateConn(guild_id)
	d.conn = conn

	if err := conn.Open(context.Background(), vc_id, false, false); err != nil {
		panic("error connecting to voice channel: " + err.Error())
	}

	if err := conn.SetSpeaking(context.Background(), voice.SpeakingFlagMicrophone); err != nil {
		panic("error setting speaking flag: " + err.Error())
	}

	pcmProvider, err := ffmpeg.New(context.Background(), disgoorgffmpeg.WithChannels(1), disgoorgffmpeg.WithSampleRate(48000))
	if err != nil {
		panic("error creating pcm provider: " + err.Error())
	}
	defer pcmProvider.Close()

	opusEncoder, _ := opus.NewEncoder(48000, 1, opus.ApplicationVoip)
	opusProvider, err := pcm.NewOpusProvider(opusEncoder, pcmProvider)
	if err != nil {
		panic("error creating opus provider: " + err.Error())
	}

	receiver, streamer, err := ffmpeg.NewOpusPCMReceiver()
	if err != nil {
		panic("error creating opus pcm receiver: " + err.Error())
	}
	defer receiver.Close()

	d.streamer = streamer

	conn.SetEventHandlerFunc(func(opCode voice.Opcode, data voice.GatewayMessageData) {
		switch opCode {
		case 11:
			d.streamer.Close()
			d.streamer = d.streamer.Reopen()
			d.playback.AddStream(d.streamer)
			log.Println("Reopened streamer")
		case voice.OpcodeClientDisconnect:
			d.streamer.Close()
			log.Println("Client disconnected, closing streamer")
		}
	})

	conn.SetOpusFrameReceiver(pcm.NewPCMOpusReceiver(nil, receiver, nil))
	conn.SetOpusFrameProvider(opusProvider)
	if err = pcmProvider.Wait(); err != nil {
		panic("error waiting for opus provider: " + err.Error())
	}

	closeChan <- syscall.SIGTERM
}
