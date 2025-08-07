# Golte - GSM/LTE to Discord Bridge

Golte is a robust bridge application that connects a GSM/LTE modem to Discord, allowing you to send and receive SMS messages and make voice calls through Discord commands and embeds.

## Features

- 📱 **SMS Reception**: Automatically forwards incoming SMS messages to Discord via embeds
- 📞 **Voice Calls**: Make and receive voice calls through GSM/LTE modem with Discord notifications
- 🎯 **Discord Commands**: Send SMS messages and manage calls using Discord slash commands
- 🔧 **Robust Configuration**: YAML configuration files with environment variable support
- 📊 **Structured Logging**: Configurable logging with JSON or text output
- 🔄 **Signal Monitoring**: Automatic signal quality monitoring
- 🛡️ **Graceful Shutdown**: Clean shutdown handling with signal interception
- 🔧 **CLI Interface**: Full command-line interface with Cobra

## Installation

### Prerequisites

- Go 1.19 or later
- GSM/LTE modem connected via serial port
- Discord bot token and channel ID

### Building from Source

```bash
git clone <repository-url>
cd golte
go mod download
go build -o golte
```

## Configuration

Golte supports multiple configuration methods:

### 1. Configuration File

Create a `config.yaml` file:

```yaml
modem:
  device: "/dev/serial0"
  baud: 115200
  timeout: "20s"

discord:
  token: "your_discord_bot_token"
  channel_id: "your_discord_channel_id"
  guild_id: "your_discord_guild_id"
  voice_channel_id: "your_discord_voice_channel_id"

logging:
  level: "info"
  format: "text"
```

### 2. Environment Variables

All configuration options can be set via environment variables with the `GOLTE_` prefix:

```bash
export GOLTE_DISCORD_TOKEN="your_discord_bot_token"
export GOLTE_DISCORD_CHANNEL_ID="your_discord_channel_id"
export GOLTE_DISCORD_GUILD_ID="your_discord_guild_id"
export GOLTE_DISCORD_VOICE_CHANNEL_ID="your_discord_voice_channel_id"
export GOLTE_MODEM_DEVICE="/dev/ttyUSB0"
export GOLTE_LOGGING_LEVEL="debug"
```

### 3. Command Line Flags

```bash
./golte --discord-token="your_token" --discord-channel="your_channel_id" --discord-guild="your_guild_id" --discord-voice-channel="your_voice_channel_id" --device="/dev/ttyUSB0"
```

## Usage

### Running the Server

Start the main application:

```bash
./golte
```

Or with custom configuration:

```bash
./golte --config=./custom-config.yaml --verbose
```

### Available Commands

#### Start the Server (default)
```bash
./golte [flags]
```

#### Validate Configuration
```bash
./golte config validate
```

#### Show Current Configuration
```bash
./golte config show
```

#### Version Information
```bash
./golte version
```

### Command Line Options

- `--config`: Path to configuration file
- `--verbose, -v`: Enable verbose logging
- `--device, -d`: Modem device path (default: "/dev/serial0")
- `--baud, -b`: Baud rate (default: 115200)
- `--timeout`: Command timeout (default: 20s)
- `--discord-token`: Discord bot token
- `--discord-channel`: Discord channel ID for SMS messages
- `--discord-guild`: Discord guild (server) ID
- `--discord-voice-channel`: Discord voice channel ID for call notifications
- `--log-level`: Log level (debug, info, warn, error)
- `--log-format`: Log format (text, json)

## Discord Setup

### 1. Create a Discord Bot

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Go to the "Bot" section
4. Create a bot and copy the token
5. Enable the "Slash Commands" scope
6. Add the bot to your server with appropriate permissions (Send Messages, Use Slash Commands, Connect to Voice Channels)

### 2. Get Discord IDs

1. **Guild ID**: Right-click on your Discord server name and select "Copy Server ID"
2. **Channel ID**: Right-click on the text channel for SMS messages and select "Copy Channel ID"  
3. **Voice Channel ID**: Right-click on the voice channel for call notifications and select "Copy Channel ID"

Note: You may need to enable Developer Mode in Discord User Settings > Advanced > Developer Mode

## Hardware Setup

### Supported Modems

Golte works with most AT command-compatible GSM/LTE modems, including:

- Quectel series (EC25, EC21, etc.)
- SIMCom series (SIM7600, SIM800, etc.)
- u-blox series (SARA-R4, SARA-G3, etc.)

### Connection

Connect your modem to the system via:
- USB (appears as `/dev/ttyUSB0` or similar)
- UART/Serial (typically `/dev/serial0` on Raspberry Pi)

## Discord Commands

Once running, the following slash commands are available in Discord:

### `/send`
Send an SMS message through the modem.

**Options:**
- `number`: Phone number to send to (required)
- `message`: Message content (required)

**Example:**
```
/send number:+1234567890 message:Hello from Discord!
```

### `/call`
Initiate a voice call through the modem.

**Options:**
- `number`: Phone number to call (required)

**Example:**
```
/call number:+1234567890
```

### `/hangup`
Hang up the current active call.

**Example:**
```
/hangup
```

## Call Features

### Incoming Calls
- Automatically detects incoming voice calls
- Sends notifications to Discord with caller ID
- Automatically answers incoming calls
- Supports caller line identification (CLIP)

### Outgoing Calls  
- Initiate calls through Discord slash commands
- Real-time call status notifications
- Manual call termination via Discord commands

### Call Management
- Mute/unmute microphone during calls
- Voice mute control
- Call status monitoring
- Multiple call handling support

## Logging

Golte provides structured logging with configurable levels and formats:

### Log Levels
- `debug`: Detailed debugging information
- `info`: General information (default)
- `warn`: Warning messages
- `error`: Error messages only

### Log Formats
- `text`: Human-readable text format
- `json`: Structured JSON format (useful for log aggregation)

### Example JSON Log Output
```json
{"time":"2024-01-15T10:30:45Z","level":"INFO","msg":"SMS sent successfully","component":"machine","number":"+1234567890"}
```

## Error Handling

Golte includes comprehensive error handling:

- **Configuration Validation**: Validates all configuration on startup
- **Connection Recovery**: Monitors modem connection health
- **Graceful Shutdown**: Handles SIGINT/SIGTERM for clean shutdown
- **Error Propagation**: Structured error reporting with context

## Monitoring

### Signal Quality
The application automatically monitors GSM signal quality every minute and logs the results.

### Health Checks
Monitor the application health by:
- Checking log output for errors
- Monitoring the process status
- Testing SMS functionality periodically

## Development

### Project Structure

```
golte/
├── cmd/           # Cobra CLI commands
├── config/        # Configuration management
├── logger/        # Logging utilities
├── machine/       # Core machine logic
├── call/          # Voice call management and AT commands
├── main.go        # Application entry point
├── go.mod         # Go module definition
└── config.yaml.example  # Example configuration
```

### Building with Version Information

```bash
go build -ldflags="-X 'golte/cmd.Version=v1.0.0' -X 'golte/cmd.GitCommit=$(git rev-parse HEAD)' -X 'golte/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o golte
```

## Troubleshooting

### Common Issues

1. **Permission Denied on Serial Device**
   ```bash
   sudo usermod -a -G dialout $USER
   # Then logout and login again
   ```

2. **Modem Not Responding**
   - Check device path: `ls /dev/tty*`
   - Verify baud rate with modem documentation
   - Ensure no other applications are using the device

3. **Discord Commands Not Working**
   - Verify bot token is correct
   - Check bot permissions in Discord server (Send Messages, Use Slash Commands, Connect to Voice Channels)
   - Ensure slash commands are registered
   - Verify guild ID and channel IDs are correct

4. **SMS Not Being Received**
   - Check SIM card is inserted and activated
   - Verify signal strength
   - Check modem logs for AT command errors

5. **Call Issues**
   - Verify voice channel ID is correct
   - Check modem supports voice calls (AT+CLIP command)
   - Ensure proper audio hardware connection
   - Check for conflicting applications using the modem

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
./golte --log-level=debug
```

Or set via environment:

```bash
GOLTE_LOGGING_LEVEL=debug ./golte
```
