package machine

import (
	"context"
	"fmt"
	"log/slog"

	"golte/config"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

// DiscordManager handles all Discord bot operations
type DiscordManager struct {
	config     *config.Config
	client     bot.Client
	logger     *slog.Logger
	smsFunc    func(number, message string) error
	notifyFunc func(from, message string)
}

// NewDiscordManager creates a new DiscordManager instance
func NewDiscordManager(cfg *config.Config, smsFunc func(number, message string) error, notifyFunc func(from, message string)) *DiscordManager {
	return &DiscordManager{
		config:     cfg,
		logger:     slog.With("component", "discord"),
		smsFunc:    smsFunc,
		notifyFunc: notifyFunc,
	}
}

// Initialize sets up the Discord bot client
func (d *DiscordManager) Initialize() error {
	d.logger.Info("Initializing Discord client")

	client, err := disgo.New(d.config.Discord.Token,
		bot.WithDefaultGateway(),
		bot.WithEventListenerFunc(d.commandListener),
	)
	if err != nil {
		return fmt.Errorf("failed to create Discord client: %w", err)
	}

	d.client = client

	// Register commands
	if _, err = client.Rest().SetGlobalCommands(client.ApplicationID(), d.getCommands()); err != nil {
		return fmt.Errorf("failed to register Discord commands: %w", err)
	}

	d.logger.Info("Discord client initialized successfully")
	return nil
}

// Start opens the Discord gateway connection
func (d *DiscordManager) Start(ctx context.Context) error {
	if err := d.client.OpenGateway(ctx); err != nil {
		return fmt.Errorf("failed to connect to Discord gateway: %w", err)
	}
	return nil
}

// Stop closes the Discord connection
func (d *DiscordManager) Stop() {
	if d.client != nil {
		d.client.Close(context.Background())
	}
}

// getCommands returns the Discord slash commands
func (d *DiscordManager) getCommands() []discord.ApplicationCommandCreate {
	return []discord.ApplicationCommandCreate{
		discord.SlashCommandCreate{
			Name:        "send",
			Description: "sends a SMS",
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
}

// commandListener handles Discord slash commands
func (d *DiscordManager) commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	if data.CommandName() == "send" {
		phoneNumber := data.String("number")
		message := data.String("message")

		d.logger.Info("Received SMS command from Discord",
			slog.String("number", phoneNumber),
			slog.String("user", event.User().Username))

		err := d.smsFunc(phoneNumber, message)
		if err != nil {
			d.logger.Error("Failed to send SMS via Discord command",
				slog.String("number", phoneNumber),
				slog.Any("error", err))

			err = event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("SMS has **not** been sent: %v", err).
				SetEphemeral(true).
				Build())
			if err != nil {
				d.logger.Error("Failed to send Discord response", slog.Any("error", err))
			}
			return
		}

		err = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("SMS Sent!").
			SetEphemeral(true).
			Build())
		if err != nil {
			d.logger.Error("Failed to send Discord response", slog.Any("error", err))
		}

		// Notify about outgoing SMS
		if d.notifyFunc != nil {
			d.notifyFunc(fmt.Sprintf("To %s", phoneNumber), message)
		}
	}
}
