# ticketmax

ESC/POS thermal receipt printer CLI. Print markdown files over TCP — designed
for RONGTA printers, works with any ESC/POS compatible device on port 9100.

## Quick Start

```bash
# Build
go build -o thermal-printer .

# Set your printer address (or pass -addr each time)
export PRINTER_ADDR=192.168.1.100:9100

# Check printer connectivity
thermal-printer -status

# Print a test receipt (exercises all formatting)
thermal-printer -test

# Print a markdown file
thermal-printer examples/morning-report.md

# Pipe from stdin
echo "# Hello World" | thermal-printer -
```

## Usage

```
thermal-printer [flags] <file.md | ->
```

Read from a file or pipe markdown to stdin with `-`.

### Flags

| Flag | Default | Description |
|---|---|---|
| `-addr` | `127.0.0.1:9100` | Printer host:port (env: `PRINTER_ADDR`) |
| `-timeout` | `5s` | Connection timeout |
| `-width` | `42` | Paper width in characters (1–120) |
| `-spacing` | `20` | Line spacing in printer units (0–255) |
| `-test` | | Print test receipt and exit |
| `-status` | | Check connectivity and exit |
| `-version` | | Print version and exit |

### Line Spacing

The `-spacing` flag controls line density. Each unit is ~1/180 inch.

| Value | Density |
|---|---|
| `0` | Printer default |
| `16` | Very tight (~2.3 mm) |
| `20` | Tight (~2.8 mm) — **default** |
| `30` | Normal (~4.2 mm) |

## Markdown Syntax

| Markdown | Printer Output |
|---|---|
| `# Heading` | Bold, double-size, centered |
| `## Heading` | Bold, double-width, centered |
| `### Heading` | Bold |
| `**bold text**` | Bold line |
| `<u>text</u>` | Underlined line |
| `---` | Separator |
| `\| A \| B \|` | Two columns |
| `\| A \| B \| C \|` | Three columns |
| `![alt](path)` | Print image (PNG, JPEG, GIF) |
| `` ```qr `` | QR code block |
| `<!-- cut -->` | Cut paper |
| `<!-- beep -->` | Buzzer (1 beep) |
| `<!-- beep N -->` | Buzzer (N beeps) |

### Example Receipt

```markdown
# Morning Report

## Sales

| Item | Revenue |
| --- | --- |
| Widget A | $1,234 |
| Gadget B | $567 |

---

**Low stock: Widget A (3 units)**

```qr
https://dashboard.example.com
```

<!-- cut -->
```

## Claude Code Skill

ticketmax includes a Claude Code skill definition (`SKILL.md`) so Claude can
print receipts directly. Any tool that writes markdown can print:

```bash
# From a script
your-app export --format=md | thermal-printer -

# From a cron job
thermal-printer /path/to/daily-report.md
```

## Examples

See `examples/` for ready-to-use templates:

- `morning-report.md` — daily sales, inventory alerts, QR dashboard link
- `sales-receipt.md` — customer receipt with totals
- `shift-summary.md` — hourly breakdown with handoff notes

## Building

```bash
make build              # current platform
make all                # linux, windows, macOS (amd64 + arm64)
make test               # run tests
make build VERSION=1.0  # set version string
```

## Supported Printers

- RONGTA ESC/POS compatible models
- Any printer supporting standard ESC/POS over TCP port 9100
