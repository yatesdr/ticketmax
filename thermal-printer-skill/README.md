# ticketmax — thermal receipt printer CLI

Print markdown files to an ESC/POS thermal receipt printer. Designed for RONGTA printers, works with any ESC/POS compatible device.

## Quick Start

```bash
cd thermal-printer-skill
go build -o thermal-printer

# Check printer connectivity
./thermal-printer -addr=192.168.1.100:9100 -status

# Print a test receipt
./thermal-printer -addr=192.168.1.100:9100 -test

# Print a markdown file
./thermal-printer -addr=192.168.1.100:9100 examples/morning-report.md

# Pipe from stdin
cat report.md | ./thermal-printer -addr=192.168.1.100:9100 -
```

## Configuration

Set environment variables to avoid passing flags every time:

```bash
export PRINTER_ADDR=192.168.1.100:9100
./thermal-printer examples/morning-report.md
```

## Markdown Syntax

| Markdown | Printer Output |
|----------|---------------|
| `# Heading` | Bold, double-size, centered |
| `## Heading` | Bold, double-width, centered |
| `### Heading` | Bold |
| `**bold text**` | Bold line |
| `<u>text</u>` | Underlined line |
| `---` | Separator |
| `\| A \| B \|` | Two columns |
| `\| A \| B \| C \|` | Three columns |
| `![alt](path)` | Print image |
| `` ```qr `` | QR code |
| `<!-- cut -->` | Cut paper |
| `<!-- beep -->` | Buzzer (1 beep) |
| `<!-- beep N -->` | Buzzer (N beeps) |

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-addr` | Printer IP:port | `127.0.0.1:9100` (env: `PRINTER_ADDR`) |
| `-timeout` | Connection timeout | `5s` |
| `-width` | Paper width in characters | `42` |
| `-spacing` | Line spacing in printer units | `20` (tight) |
| `-test` | Print test receipt and exit | |
| `-status` | Check connectivity and exit | |

### Line Spacing

The `-spacing` flag controls line density. Each unit is 1/180 inch on most printers.

```bash
./thermal-printer -spacing=20 report.md   # tight (default)
./thermal-printer -spacing=16 report.md   # very tight
./thermal-printer -spacing=30 report.md   # normal
./thermal-printer -spacing=0  report.md   # printer default
```

## Integration

Any software that writes markdown can print receipts:

```bash
# From a script
your-app export --format=md | ./thermal-printer -

# From a cron job
./thermal-printer /path/to/daily-report.md

# As an OpenClaw skill
@bot write a morning report and print it
```

## Examples

See `examples/` for ready-to-use templates:

- `morning-report.md` — daily sales, inventory alerts, QR dashboard link
- `sales-receipt.md` — customer receipt with totals
- `shift-summary.md` — hourly breakdown with handoff notes

## Supported Printers

- RONGTA ESC/POS compatible models
- Any printer supporting standard ESC/POS protocol over TCP port 9100
