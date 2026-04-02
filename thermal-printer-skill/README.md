# Thermal Printer Skill

A Go-based OpenClaw skill for printing morning reports to a RONGTA thermal receipt printer.

## Quick Start

### 1. Build the application

```bash
go mod download
go build -o thermal-printer
```

### 2. Find your printer's IP

The RONGTA printer will print its network configuration on startup. Find the IP address (usually `192.168.x.x`).

Alternatively, scan your network:
```bash
arp-scan --localnet | grep -i rongta
```

### 3. Test the connection

```bash
./thermal-printer -type=network -addr=192.168.1.100:9100 \
  -title="Test" \
  -content="If you see this, printer works!"
```

## Installation as OpenClaw Skill

1. Copy this directory to your OpenClaw workspace:
   ```bash
   cp -r thermal-printer-skill ~/.openclaw/skills/thermal-printer
   # or
   cp -r thermal-printer-skill ~/my-workspace/skills/thermal-printer
   ```

2. Build the binary in the skill directory:
   ```bash
   cd ~/.openclaw/skills/thermal-printer
   go build -o thermal-printer
   ```

3. Your bot can now use it:
   ```
   @bot print a morning report to the thermal printer with these sales figures...
   ```

## Configuration

Set environment variables for your printer:

```bash
export PRINTER_TYPE=network
export PRINTER_ADDR=192.168.1.100:9100
```

Or pass them directly to the command:
```bash
./thermal-printer -type=network -addr=192.168.1.100:9100 -content="Report text"
```

## Advanced Usage

### Print formatted reports

Create a report file and print it:
```bash
./thermal-printer -type=network -addr=192.168.1.100:9100 \
  -title="Daily Sales" \
  -content="sales_report.txt"
```

### Generate reports from the bot

The bot can generate content and pipe it to the printer. Example from OpenClaw:
```
Generate today's sales summary and print it to the thermal printer
```

## Supported Printers

- RONGTA models with network/USB/serial connectivity
- Any ESC/POS compatible thermal receipt printer

## Troubleshooting

### Printer not found
- Verify IP address: ping the printer IP
- Check firewall: ensure port 9100 is accessible
- Test with: `nc -zv 192.168.1.100 9100`

### Nothing prints
- Check paper is loaded
- Verify printer is powered on and connected to network
- Try a simple test: `./thermal-printer -list`

### Permission issues (serial)
```bash
sudo usermod -a -G dialout $USER
# Then logout and login again
```

## ESC/POS Protocol

The printer uses the ESC/POS protocol. Key commands:
- `ESC @` - Initialize
- `ESC !` - Select print mode (size, bold, etc)
- `GS V` - Cut paper
- `LF` - Line feed

Extend `printer.go` to add more formatting options.
