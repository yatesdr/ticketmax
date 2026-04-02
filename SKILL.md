---
name: ticketmax
description: Print markdown files to a RONGTA ESC/POS thermal receipt printer. Supports headings, bold, tables, images, and QR codes.
user-invocable: true
command-dispatch: tool
---

# Thermal Printer Skill

Print markdown files to a thermal receipt printer.

## Usage

```bash
# Print a markdown file
ticketmax report.md

# Pipe from stdin
cat report.md | ticketmax -

# Generate and print on the fly
echo "# Alert\n\n**Server down**" | ticketmax -

# Check printer connectivity
ticketmax -status

# Print a test receipt
ticketmax -test
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-addr` | `127.0.0.1:9100` | Printer host:port (env: `PRINTER_ADDR`) |
| `-timeout` | `5s` | Connection timeout |
| `-width` | `42` | Paper width in characters (1–120) |
| `-spacing` | `20` | Line spacing in printer units (0–255) |
| `-test` | | Print test receipt and exit |
| `-status` | | Check connectivity and exit |

## Markdown Syntax

| Markdown | Printer Output |
|----------|---------------|
| `# Heading` | Bold, double-size, centered |
| `## Heading` | Bold, double-width, centered |
| `### Heading` | Bold |
| `**bold text**` | Bold line |
| `<u>text</u>` | Underlined line |
| `---` | Separator line |
| `\| A \| B \|` | Two columns |
| `\| A \| B \| C \|` | Three columns |
| `![alt](path)` | Print image |
| `` ```qr `` | QR code block |
| `<!-- cut -->` | Cut paper |
| `<!-- beep -->` | Buzzer (1 beep) |
| `<!-- beep N -->` | Buzzer (N beeps) |

## Example Report

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
```

## Integration

Any software that can write a markdown file or pipe text to stdout can print:

```bash
# From a script
shingo export --format=md | ticketmax -

# From a cron job
ticketmax /path/to/daily-report.md

# From OpenClaw bot
@bot write a morning report in markdown and print it
```
