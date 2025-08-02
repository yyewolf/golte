package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/warthog618/modem/at"
)

var (
	token      = ""
	webhookURL = ""

	commands = []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "send",
			Description: "sends a sms",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "number",
					Description: "The phone number to send the message to",
					Required:    true,
				},
				discord.ApplicationCommandOptionString{
					Name:        "message",
					Description: "What to say",
					Required:    true,
				},
			},
		},
	}
)

func (m *Machine) commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "send" {
		// Check if it would be an SMS or a long SMS
		phoneNumber := data.String("number")
		message := data.String("message")

		var err error
		if len(message) > 160 {
			// Long SMS, split into multiple messages
			_, err = m.gsm.SendLongMessage(phoneNumber, message, at.WithTimeout(5*time.Second))
		} else {
			_, err = m.gsm.SendShortMessage(phoneNumber, message, at.WithTimeout(5*time.Second))
		}

		if err != nil {
			err = event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("SMS has **not** been sent: %v", err).
				SetEphemeral(true).
				Build(),
			)
			if err != nil {
				slog.Error("error on sending response", slog.Any("err", err))
			}
			return
		}

		err = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("SMS Sent !").
			SetEphemeral(true).
			Build(),
		)
		if err != nil {
			slog.Error("error on sending response", slog.Any("err", err))
		}
	}
}

func (m *Machine) sendDiscordMessage(from, message string) error {
	data := fmt.Sprintf(`{"username":"%s","content": "%s"}`, from, message)
	req, err := http.NewRequest("POST", webhookURL, strings.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message: %s", body)
	}

	return nil
}
