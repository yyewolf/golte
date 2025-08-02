# Service Installation

This guide explains how to install and run Golte as a systemd service on Linux.

## Quick Installation

Run the automated installation script as root:

```bash
sudo ./install-service.sh
```

This script will:
- Create a dedicated `golte` user
- Build the binary
- Install files to `/opt/golte/`
- Configure systemd service
- Set up proper permissions for serial device access

## Manual Installation

If you prefer to install manually:

### 1. Create Service User

```bash
sudo useradd --system --no-create-home --shell /bin/false --group dialout golte
```

### 2. Create Installation Directory

```bash
sudo mkdir -p /opt/golte
sudo chown golte:dialout /opt/golte
```

### 3. Build and Install Binary

```bash
go build -o golte -ldflags "-s -w" .
sudo cp golte /opt/golte/
sudo chown golte:dialout /opt/golte/golte
sudo chmod 755 /opt/golte/golte
```

### 4. Install Configuration

```bash
sudo cp config.yaml /opt/golte/  # or config.yaml.example as config.yaml
sudo chown golte:dialout /opt/golte/config.yaml
sudo chmod 644 /opt/golte/config.yaml
```

### 5. Install Service

```bash
sudo cp golte.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 6. Configure Serial Device Permissions

Create udev rules for modem access:

```bash
sudo tee /etc/udev/rules.d/99-golte-modem.rules << EOF
SUBSYSTEM=="tty", ATTRS{idVendor}=="*", ATTRS{idProduct}=="*", GROUP="dialout", MODE="0664"
KERNEL=="ttyUSB*", GROUP="dialout", MODE="0664"
KERNEL=="ttyACM*", GROUP="dialout", MODE="0664"
KERNEL=="serial*", GROUP="dialout", MODE="0664"
EOF

sudo udevadm control --reload-rules
sudo udevadm trigger
```

## Service Management

### Enable and Start Service

```bash
sudo systemctl enable golte
sudo systemctl start golte
```

### Check Service Status

```bash
sudo systemctl status golte
```

### View Logs

```bash
# Follow live logs
sudo journalctl -u golte -f

# View recent logs
sudo journalctl -u golte -n 50

# View logs with timestamps
sudo journalctl -u golte --since "1 hour ago"
```

### Stop and Disable Service

```bash
sudo systemctl stop golte
sudo systemctl disable golte
```

## Configuration

Edit the configuration file at `/opt/golte/config.yaml`:

```bash
sudo nano /opt/golte/config.yaml
```

After changing the configuration, restart the service:

```bash
sudo systemctl restart golte
```

## Troubleshooting

### Service Won't Start

1. Check service status and logs:
   ```bash
   sudo systemctl status golte
   sudo journalctl -u golte -n 50
   ```

2. Verify configuration:
   ```bash
   /opt/golte/golte --config /opt/golte/config.yaml --help
   ```

3. Test manually:
   ```bash
   sudo -u golte /opt/golte/golte --config /opt/golte/config.yaml
   ```

### Permission Issues

1. Check file ownership:
   ```bash
   ls -la /opt/golte/
   ```

2. Verify user is in dialout group:
   ```bash
   groups golte
   ```

3. Check device permissions:
   ```bash
   ls -la /dev/ttyUSB* /dev/ttyACM* /dev/serial*
   ```

### Serial Device Not Found

1. List available serial devices:
   ```bash
   ls -la /dev/tty{USB,ACM}*
   ```

2. Check if device is recognized:
   ```bash
   dmesg | grep -i usb
   lsusb
   ```

3. Update device path in configuration:
   ```bash
   sudo nano /opt/golte/config.yaml
   ```

## Uninstallation

To completely remove the service:

```bash
# Stop and disable service
sudo systemctl stop golte
sudo systemctl disable golte

# Remove service file
sudo rm /etc/systemd/system/golte.service
sudo systemctl daemon-reload

# Remove installation directory
sudo rm -rf /opt/golte

# Remove user
sudo userdel golte

# Remove udev rules
sudo rm /etc/udev/rules.d/99-golte-modem.rules
sudo udevadm control --reload-rules
```
