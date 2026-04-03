# ticketmax

Print markdown files to ESC/POS thermal receipt printers over TCP. Headings,
bold, underline, tables, images, QR codes, paper cuts, and buzzer — all from
standard `.md` files.

Works with any ESC/POS printer on port 9100 (tested with Rongta).

## Quick Start

```bash
# Install
go install github.com/yatesdr/ticketmax@latest

# Or build from source
git clone https://github.com/yatesdr/ticketmax.git
cd ticketmax
make build
```

```bash
# Set your printer address (or pass -addr each time)
export PRINTER_ADDR=192.168.1.100:9100

# Check connectivity
ticketmax -status

# Print a test receipt
ticketmax -test

# Print a markdown file
ticketmax examples/morning-report.md

# Pipe from stdin
echo "# Hello World" | ticketmax -
```

## Usage

```
ticketmax [flags] <file.md | ->
```

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
| `\| A \| B \|` | Two columns (right-aligned values) |
| `\| A \| B \| C \|` | Three columns |
| `![alt](path)` | Image (PNG, JPEG, GIF) |
| `` ```qr `` | QR code block |
| `<!-- cut -->` | Cut paper |
| `<!-- beep -->` | Buzzer (1 beep) |
| `<!-- beep N -->` | Buzzer (N beeps) |

### Example

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

## Agent Skill

ticketmax includes a skill definition (`SKILL.md`) so it can be used by
Claude Code agents and OpenClaw bots. Any tool that writes markdown can print
to a receipt printer:

```bash
# Pipe output from another tool
your-app export --format=md | ticketmax -

# Cron job
ticketmax /path/to/daily-report.md

# OpenClaw bot
@bot write a morning report in markdown and print it
```

## Examples

See `examples/` for ready-to-use templates:

- `morning-report.md` — daily sales, inventory alerts, QR dashboard link
- `sales-receipt.md` — customer receipt with totals
- `shift-summary.md` — hourly breakdown with handoff notes
- `feature-test.md` — exercises every supported markdown feature

## Building

```bash
make build              # current platform
make all                # linux, windows, macOS (amd64 + arm64)
make test               # run tests
make build VERSION=1.0  # set version string
```
