package main

import (
	"context"
	"flag"
	"io"
	"log"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/warthog618/modem/at"
	"github.com/warthog618/modem/gsm"
	"github.com/warthog618/modem/serial"
	"github.com/warthog618/modem/trace"
)

func main() {
	dev := flag.String("d", "/dev/serial0", "path to modem device")
	baud := flag.Int("b", 115200, "baud rate")
	timeout := flag.Duration("t", 20*time.Second, "command timeout period")

	flag.Parse()
	m, err := serial.New(serial.WithPort(*dev), serial.WithBaud(*baud))
	if err != nil {
		log.Println(err)
		return
	}
	defer m.Close()
	var mio io.ReadWriter = m
	mio = trace.New(m, trace.WithReadFormat("r: %v"))
	log.Printf("Using modem device %s at %d baud\n", *dev, *baud)
	g := gsm.New(at.New(mio, at.WithTimeout(*timeout), at.WithCmds("I")))
	err = g.Init()
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("Modem initialized\n")

	go pollSignalQuality(g, timeout)

	self := &Machine{
		gsm: g,
	}

	err = g.StartMessageRx(
		func(msg gsm.Message) {
			log.Printf("%s: %s\n", msg.Number, msg.Message)
			self.sendDiscordMessage(msg.Number, msg.Message)
		},
		func(err error) {
			log.Printf("err: %v\n", err)
		})
	if err != nil {
		log.Println(err)
		return
	}
	defer g.StopMessageRx()

	slog.Info("starting example...")
	slog.Info("disgo version", slog.String("version", disgo.Version))

	client, err := disgo.New(token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(self.commandListener),
	)
	if err != nil {
		slog.Error("error while building disgo instance", slog.Any("err", err))
		return
	}

	defer client.Close(context.TODO())

	if _, err = client.Rest().SetGlobalCommands(client.ApplicationID(), commands); err != nil {
		slog.Error("error while registering commands", slog.Any("err", err))
	}

	if err = client.OpenGateway(context.TODO()); err != nil {
		slog.Error("error while connecting to gateway", slog.Any("err", err))
	}

	select {
	case <-g.Closed():
		log.Fatal("modem closed, exiting...")
	}
}

// pollSignalQuality polls the modem to read signal quality every minute.
//
// This is run in parallel to SMS reception to demonstrate separate goroutines
// interacting with the modem.
func pollSignalQuality(g *gsm.GSM, timeout *time.Duration) {
	for {
		select {
		case <-time.After(time.Minute):
			i, err := g.Command("+CSQ")
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("Signal quality: %v\n", i)
			}
		case <-g.Closed():
			return
		}
	}
}
