package machine

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"golte/config"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/snowflake/v2"
)

// DiscordManager handles all Discord bot operations
type DiscordManager struct {
	config     *config.Config
	client     bot.Client
	logger     *slog.Logger
	smsFunc    func(number, message string) error
	callFunc   func(number string) error
	hangupFunc func() error
	notifyFunc func(notificationType NotificationType, from, message string)
}

// NewDiscordManager creates a new DiscordManager instance
func NewDiscordManager(cfg *config.Config, smsFunc func(number, message string) error, callFunc func(number string) error, hangupFunc func() error, notifyFunc func(notificationType NotificationType, from, message string)) *DiscordManager {
	return &DiscordManager{
		config:     cfg,
		logger:     slog.With("component", "discord"),
		smsFunc:    smsFunc,
		callFunc:   callFunc,
		hangupFunc: hangupFunc,
		notifyFunc: notifyFunc,
	}
}

// Initialize sets up the Discord bot client
func (d *DiscordManager) Initialize() error {
	d.logger.Info("Initializing Discord client")

	client, err := disgo.New(d.config.Discord.Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentMessageContent|gateway.IntentGuilds|gateway.IntentGuildMessages|gateway.IntentDirectMessages|gateway.IntentGuildVoiceStates),
		),
		bot.WithEventListenerFunc(d.commandListener),
		bot.WithEventListenerFunc(d.messageListener),
		bot.WithEventListenerFunc(d.readyListener),
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
		discord.SlashCommandCreate{
			Name:        "call",
			Description: "makes a phone call",
			Options: []discord.ApplicationCommandOption{
				discord.ApplicationCommandOptionString{
					Name:        "number",
					Description: "The phone number to call",
					Required:    true,
				},
			},
		},
		discord.SlashCommandCreate{
			Name:        "hangup",
			Description: "hangs up the current phone call",
		},
	}
}

// commandListener handles Discord slash commands
func (d *DiscordManager) commandListener(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()

	switch data.CommandName() {
	case "send":
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
			d.notifyFunc(NotificationTypeSMS, fmt.Sprintf("To %s", phoneNumber), message)
		}

	case "call":
		phoneNumber := data.String("number")

		d.logger.Info("Received call command from Discord",
			slog.String("number", phoneNumber),
			slog.String("user", event.User().Username))

		err := d.callFunc(phoneNumber)
		if err != nil {
			d.logger.Error("Failed to start call via Discord command",
				slog.String("number", phoneNumber),
				slog.Any("error", err))

			err = event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Call has **not** been started: %v", err).
				SetEphemeral(true).
				Build())
			if err != nil {
				d.logger.Error("Failed to send Discord response", slog.Any("error", err))
			}
			return
		}

		err = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("📞 Calling %s...", phoneNumber).
			SetEphemeral(true).
			Build())
		if err != nil {
			d.logger.Error("Failed to send Discord response", slog.Any("error", err))
		}

		// Notify about outgoing call
		if d.notifyFunc != nil {
			d.notifyFunc(NotificationTypeCall, fmt.Sprintf("Calling %s", phoneNumber), "📞 Call initiated")
		}

	case "hangup":
		d.logger.Info("Received hangup command from Discord",
			slog.String("user", event.User().Username))

		err := d.hangupFunc()
		if err != nil {
			d.logger.Error("Failed to hang up call via Discord command",
				slog.Any("error", err))

			err = event.CreateMessage(discord.NewMessageCreateBuilder().
				SetContentf("Call has **not** been hung up: %v", err).
				SetEphemeral(true).
				Build())
			if err != nil {
				d.logger.Error("Failed to send Discord response", slog.Any("error", err))
			}
			return
		}

		err = event.CreateMessage(discord.NewMessageCreateBuilder().
			SetContent("📞 Call hung up!").
			SetEphemeral(true).
			Build())
		if err != nil {
			d.logger.Error("Failed to send Discord response", slog.Any("error", err))
		}

		// Notify about call hangup
		if d.notifyFunc != nil {
			d.notifyFunc(NotificationTypeCall, "Call ended", "📞 Call hung up")
		}
	}
}

// readyListener handles Discord ready event
func (d *DiscordManager) readyListener(event *events.Ready) {
	d.logger.Info("Discord bot is ready, connecting to voice channel")

	go func() {
		var ch = make(chan os.Signal, 1)
		d.ConnectAndPlay(ch)
	}()
}

// messageListener handles Discord message events for replying to SMS embeds
func (d *DiscordManager) messageListener(event *events.MessageCreate) {
	log.Printf("Received message from Discord: %s", event.Message.Content)
	// Ignore bot messages and messages not in the configured channel
	if event.Message.Author.Bot || event.Message.ChannelID.String() != d.config.Discord.ChannelID {
		return
	}

	// Check if this message is a reply
	if event.Message.MessageReference == nil {
		d.logger.Info("Message is not a reply, ignoring")
		return
	}

	// Get the referenced message
	referencedMessage, err := d.client.Rest().GetMessage(event.Message.ChannelID, *event.Message.MessageReference.MessageID)
	if err != nil {
		d.logger.Debug("Failed to get referenced message", slog.Any("error", err))
		return
	}

	// Check if the referenced message is from our bot and has an SMS embed
	if referencedMessage.Author.ID != d.client.ApplicationID() || len(referencedMessage.Embeds) == 0 {
		d.logger.Info("Referenced message is not from bot or has no embeds, ignoring")
		return
	}

	// Find the SMS embed by checking for the SMS title
	var smsEmbed *discord.Embed
	for _, embed := range referencedMessage.Embeds {
		if embed.Title == "📱 SMS Message" {
			smsEmbed = &embed
			break
		}
	}

	if smsEmbed == nil {
		d.logger.Info("No SMS embed found in referenced message")
		return
	}

	// Extract phone number from the embed's author field (which contains the phone number)
	phoneNumber := smsEmbed.Author.Name
	if phoneNumber == "" {
		d.logger.Info("No phone number found in SMS embed")
		return
	}

	replyMessage := event.Message.Content
	if replyMessage == "" {
		d.logger.Info("Empty reply message")
		return
	}

	d.logger.Info("Received SMS reply from Discord",
		slog.String("number", phoneNumber),
		slog.String("user", event.Message.Author.Username),
		slog.String("message", replyMessage))

	// Send the SMS
	err = d.smsFunc(phoneNumber, replyMessage)
	if err != nil {
		d.logger.Error("Failed to send SMS reply",
			slog.String("number", phoneNumber),
			slog.Any("error", err))

		// React with an error emoji
		err = d.client.Rest().AddReaction(event.Message.ChannelID, event.Message.ID, "❌")
		if err != nil {
			d.logger.Error("Failed to add error reaction", slog.Any("error", err))
		}
		return
	}

	// React with a checkmark to confirm SMS was sent
	err = d.client.Rest().AddReaction(event.Message.ChannelID, event.Message.ID, "✅")
	if err != nil {
		d.logger.Error("Failed to add success reaction", slog.Any("error", err))
	}
}

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeSMS  NotificationType = "sms"
	NotificationTypeCall NotificationType = "call"
)

// SendEmbed sends an embed message to the configured Discord channel
func (d *DiscordManager) SendEmbed(notificationType NotificationType, from, message string) error {
	channelID, err := snowflake.Parse(d.config.Discord.ChannelID)
	if err != nil {
		return fmt.Errorf("invalid channel ID: %w", err)
	}

	var embed discord.Embed
	switch notificationType {
	case NotificationTypeSMS:
		embed = discord.NewEmbedBuilder().
			SetTitle("📱 SMS Message").
			SetDescription(message).
			SetAuthor(from, "", "").
			SetColor(0x00ff00).
			SetTimestamp(time.Now()).
			Build()
	case NotificationTypeCall:
		embed = discord.NewEmbedBuilder().
			SetTitle("📞 Call").
			SetDescription(message).
			SetAuthor(from, "", "").
			SetColor(0x0099ff).
			SetTimestamp(time.Now()).
			Build()
	default:
		return fmt.Errorf("unsupported notification type: %s", notificationType)
	}

	_, err = d.client.Rest().CreateMessage(channelID, discord.NewMessageCreateBuilder().
		SetEmbeds(embed).
		Build())

	if err != nil {
		d.logger.Error("Failed to send embed to Discord",
			slog.String("type", string(notificationType)),
			slog.String("from", from),
			slog.String("channel", d.config.Discord.ChannelID),
			slog.Any("error", err))
		return fmt.Errorf("failed to send Discord message: %w", err)
	}

	d.logger.Debug("Sent embed to Discord",
		slog.String("type", string(notificationType)),
		slog.String("from", from),
		slog.String("channel", d.config.Discord.ChannelID))

	return nil
}
