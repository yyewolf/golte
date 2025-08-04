package machine

import (
	"context"
	"golte/ffmpg"
	"os"
	"syscall"

	"github.com/disgoorg/audio/opus"
	"github.com/disgoorg/audio/pcm"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/ffmpeg-audio"
	"github.com/disgoorg/snowflake/v2"
)

func (d *DiscordManager) ConnectAndPlay(closeChan chan os.Signal) {
	guild_id := snowflake.MustParse(d.config.Discord.GuildID)
	vc_id := snowflake.MustParse(d.config.Discord.VoiceChannelID)

	conn := d.client.VoiceManager().CreateConn(guild_id)

	if err := conn.Open(context.Background(), vc_id, false, false); err != nil {
		panic("error connecting to voice channel: " + err.Error())
	}

	if err := conn.SetSpeaking(context.Background(), voice.SpeakingFlagMicrophone); err != nil {
		panic("error setting speaking flag: " + err.Error())
	}

	pcmProvider, err := ffmpg.New(context.Background(), ffmpeg.WithChannels(1), ffmpeg.WithSampleRate(48000))
	if err != nil {
		panic("error creating pcm provider: " + err.Error())
	}
	defer pcmProvider.Close()

	opusEncoder, _ := opus.NewEncoder(48000, 1, opus.ApplicationVoip)
	opusProvider, err := pcm.NewOpusProvider(opusEncoder, pcmProvider)
	if err != nil {
		panic("error creating opus provider: " + err.Error())
	}

	pcmReceiver, err := ffmpg.NewALSAReceiver()
	if err != nil {
		panic("error creating pcm receiver: " + err.Error())
	}
	defer pcmReceiver.Close()

	conn.SetOpusFrameReceiver(pcm.NewPCMOpusReceiver(nil, pcmReceiver, nil))
	conn.SetOpusFrameProvider(opusProvider)
	if err = pcmProvider.Wait(); err != nil {
		panic("error waiting for opus provider: " + err.Error())
	}

	closeChan <- syscall.SIGTERM
}
