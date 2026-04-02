package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ElementType identifies the kind of printable unit produced by the parser.
type ElementType string

// Element types produced by the markdown parser.
const (
	elemTitle     ElementType = "title"
	elemText      ElementType = "text"
	elemBold      ElementType = "bold"
	elemSeparator ElementType = "separator"
	elemColumns   ElementType = "columns"
	elemImage     ElementType = "image"
	elemQRCode    ElementType = "qrcode"
	elemFeed      ElementType = "feed"
	elemCut       ElementType = "cut"
	elemUnderline ElementType = "underline"
	elemBeep      ElementType = "beep"
)

// Element is a single printable unit parsed from markdown.
type Element struct {
	Type      ElementType
	Text      string
	Style     TextStyle
	Values    []string // column values
	Path      string   // image path
	Data      string   // QR code data
	BeepTimes int      // number of beeps (elemBeep)
}

// ParseMarkdown converts a markdown string into a sequence of printer elements.
//
// Supported syntax:
//
//	# Heading          → bold, double-width+height, centered
//	## Heading         → bold, double-width, centered
//	### Heading        → bold
//	**bold text**      → bold line
//	---  ***  ___      → separator
//	| A | B |          → 2-column row
//	| A | B | C |      → 3-column row
//	| --- | --- |      → skipped (table header separator)
//	![alt](path)       → image
//	```qr … ```        → QR code
//	<u>text</u>        → underlined line
//	<!-- cut -->        → paper cut
//	<!-- beep -->       → buzzer (1 beep)
//	<!-- beep N -->     → buzzer (N beeps)
//	(empty line)       → blank line feed
//	anything else      → plain text
func ParseMarkdown(input string) ([]Element, error) {
	lines := strings.Split(input, "\n")
	var elems []Element

	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		trimmed := strings.TrimSpace(raw)

		switch {
		// Fenced QR code block.
		case strings.HasPrefix(trimmed, "```qr"):
			data, advance, err := parseFencedBlock(lines, i)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			i = advance
			elems = append(elems, Element{Type: elemQRCode, Data: data})

		// Skip other fenced code blocks — just print their contents as text.
		case strings.HasPrefix(trimmed, "```"):
			i++ // skip opening fence
			for i < len(lines) && strings.TrimSpace(lines[i]) != "```" {
				elems = append(elems, Element{Type: elemText, Text: lines[i]})
				i++
			}
			// i now points at closing ``` (or past end); loop increment moves past it.

		// HTML comment directives.
		case strings.Contains(trimmed, "<!-- cut -->"):
			elems = append(elems, Element{Type: elemCut})

		case strings.Contains(trimmed, "<!-- beep"):
			times := parseBeepDirective(trimmed)
			elems = append(elems, Element{Type: elemBeep, BeepTimes: times})

		// Empty line → feed.
		case trimmed == "":
			elems = append(elems, Element{Type: elemFeed})

		// H1.
		case strings.HasPrefix(trimmed, "# "):
			text := strings.TrimPrefix(trimmed, "# ")
			elems = append(elems, Element{
				Type:  elemTitle,
				Text:  text,
				Style: TextStyle{Bold: true, DoubleWidth: true, DoubleHeight: true, Centered: true},
			})

		// H2.
		case strings.HasPrefix(trimmed, "## "):
			text := strings.TrimPrefix(trimmed, "## ")
			elems = append(elems, Element{
				Type:  elemTitle,
				Text:  text,
				Style: TextStyle{Bold: true, DoubleWidth: true, Centered: true},
			})

		// H3.
		case strings.HasPrefix(trimmed, "### "):
			text := strings.TrimPrefix(trimmed, "### ")
			elems = append(elems, Element{
				Type:  elemTitle,
				Text:  text,
				Style: TextStyle{Bold: true},
			})

		// Thematic break (separator).
		case isThematicBreak(trimmed):
			elems = append(elems, Element{Type: elemSeparator})

		// Table row.
		case strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|"):
			cols := parseTableRow(trimmed)
			if cols == nil {
				continue // table separator row like | --- | --- |
			}
			elems = append(elems, Element{Type: elemColumns, Values: cols})

		// Image.
		case strings.HasPrefix(trimmed, "!["):
			path := parseImageLine(trimmed)
			if path != "" {
				elems = append(elems, Element{Type: elemImage, Path: path})
			} else {
				// Malformed image syntax — print as text.
				elems = append(elems, Element{Type: elemText, Text: stripInlineMarkup(trimmed)})
			}

		// Full-line underline: <u>text</u>
		case isFullLineUnderline(trimmed):
			text := trimmed[3 : len(trimmed)-4]
			elems = append(elems, Element{Type: elemUnderline, Text: text})

		// Full-line bold.
		case isFullLineBold(trimmed):
			text := trimmed[2 : len(trimmed)-2]
			elems = append(elems, Element{Type: elemBold, Text: text})

		// Regular text — strip any remaining inline markers.
		default:
			elems = append(elems, Element{Type: elemText, Text: stripInlineMarkup(trimmed)})
		}
	}

	return elems, nil
}

// PrintMarkdown parses markdown and sends it to the printer.
// Consecutive blank lines are collapsed to save paper.
// A paper cut is always appended at the end.
func (p *Printer) PrintMarkdown(input string) error {
	elems, err := ParseMarkdown(input)
	if err != nil {
		return fmt.Errorf("parse markdown: %w", err)
	}

	// Collapse consecutive feeds to a single feed to reduce paper waste.
	elems = collapseFeeds(elems)

	if err := p.Initialize(); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	for i, el := range elems {
		if err := p.printElement(el); err != nil {
			return fmt.Errorf("element %d (%s): %w", i, el.Type, err)
		}
	}

	// Always cut at the end unless the last element was already a cut.
	if len(elems) == 0 || elems[len(elems)-1].Type != elemCut {
		if err := p.CutPaper(); err != nil {
			return fmt.Errorf("final cut: %w", err)
		}
	}

	return nil
}

// collapseFeeds merges runs of consecutive feed elements into one,
// and strips leading/trailing feeds to minimize wasted paper.
func collapseFeeds(elems []Element) []Element {
	var out []Element
	for _, el := range elems {
		if el.Type == elemFeed {
			// Skip if the previous element was also a feed, or if output is empty.
			if len(out) == 0 || out[len(out)-1].Type == elemFeed {
				continue
			}
		}
		out = append(out, el)
	}
	// Strip trailing feed (cut will provide the spacing).
	if len(out) > 0 && out[len(out)-1].Type == elemFeed {
		out = out[:len(out)-1]
	}
	return out
}

// printElement dispatches a single parsed element to the appropriate printer method.
func (p *Printer) printElement(el Element) error {
	switch el.Type {
	case elemTitle:
		return p.PrintStyledText(el.Text, el.Style)
	case elemText:
		return p.PrintLine(el.Text)
	case elemBold:
		return p.PrintBold(el.Text)
	case elemUnderline:
		return p.PrintUnderline(el.Text)
	case elemSeparator:
		return p.PrintSeparator()
	case elemColumns:
		switch len(el.Values) {
		case 2:
			return p.PrintColumns(el.Values[0], el.Values[1])
		case 3:
			return p.PrintThreeColumns(el.Values[0], el.Values[1], el.Values[2])
		default:
			return p.PrintLine(strings.Join(el.Values, "\t"))
		}
	case elemImage:
		return p.PrintImageFromFile(el.Path)
	case elemQRCode:
		return p.PrintQRCode(el.Data)
	case elemFeed:
		return p.PrintLine("")
	case elemCut:
		return p.CutPaper()
	case elemBeep:
		return p.Beep(byte(el.BeepTimes), 2)
	default:
		return fmt.Errorf("unknown element type: %s", el.Type)
	}
}

// ---------------------------------------------------------------------------
// Parsing helpers
// ---------------------------------------------------------------------------

// isThematicBreak returns true for markdown horizontal rules: ---, ***, ___.
func isThematicBreak(s string) bool {
	if len(s) < 3 {
		return false
	}
	clean := strings.ReplaceAll(s, " ", "")
	if len(clean) < 3 {
		return false
	}
	ch := clean[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	for i := 1; i < len(clean); i++ {
		if clean[i] != ch {
			return false
		}
	}
	return true
}

// parseTableRow extracts cell values from a markdown table row.
// Returns nil for separator rows (| --- | --- |).
func parseTableRow(line string) []string {
	// Trim leading/trailing pipes.
	inner := strings.TrimSpace(line)
	inner = strings.TrimPrefix(inner, "|")
	inner = strings.TrimSuffix(inner, "|")

	cells := strings.Split(inner, "|")
	var values []string
	for _, cell := range cells {
		values = append(values, strings.TrimSpace(cell))
	}

	// Detect separator rows: all cells are dashes (possibly with colons).
	isSeparator := true
	for _, v := range values {
		stripped := strings.Trim(v, ":-")
		if stripped != "" {
			isSeparator = false
			break
		}
	}
	if isSeparator {
		return nil
	}

	return values
}

// parseImageLine extracts the file path from ![alt](path).
func parseImageLine(line string) string {
	start := strings.Index(line, "](")
	if start < 0 {
		return ""
	}
	rest := line[start+2:]
	end := strings.Index(rest, ")")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// parseFencedBlock reads a ```qr fenced block and returns its content.
// Returns the content, the line index of the closing fence, and any error.
func parseFencedBlock(lines []string, start int) (string, int, error) {
	var dataLines []string
	i := start + 1
	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "```" {
			return strings.TrimSpace(strings.Join(dataLines, "\n")), i, nil
		}
		dataLines = append(dataLines, strings.TrimSpace(lines[i]))
		i++
	}
	return "", i, fmt.Errorf("unclosed ```qr block starting at line %d", start+1)
}

// isFullLineBold returns true if the entire line is wrapped in ** **.
func isFullLineBold(s string) bool {
	return len(s) > 4 && strings.HasPrefix(s, "**") && strings.HasSuffix(s, "**") &&
		!strings.Contains(s[2:len(s)-2], "**")
}

// stripInlineMarkup removes common inline markdown markers for clean printing.
func stripInlineMarkup(s string) string {
	// Bold: **text** or __text__
	s = stripPairs(s, "**")
	s = stripPairs(s, "__")
	// Italic: *text* or _text_
	s = stripPairs(s, "*")
	s = stripPairs(s, "_")
	// Inline code: `text`
	s = stripPairs(s, "`")
	// Underline: <u>text</u>
	s = stripHTMLTag(s, "u")
	return s
}

// stripHTMLTag removes matched <tag>...</tag> pairs from s.
func stripHTMLTag(s, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	for {
		start := strings.Index(s, openTag)
		if start < 0 {
			return s
		}
		end := strings.Index(s[start+len(openTag):], closeTag)
		if end < 0 {
			return s
		}
		end += start + len(openTag)
		s = s[:start] + s[start+len(openTag):end] + s[end+len(closeTag):]
	}
}

// isFullLineUnderline returns true if the entire line is <u>text</u>.
func isFullLineUnderline(s string) bool {
	return len(s) > 7 && strings.HasPrefix(s, "<u>") && strings.HasSuffix(s, "</u>") &&
		!strings.Contains(s[3:len(s)-4], "</u>")
}

// parseBeepDirective extracts the beep count from <!-- beep --> or <!-- beep N -->.
func parseBeepDirective(s string) int {
	// Extract content between <!-- and -->.
	start := strings.Index(s, "<!--")
	end := strings.Index(s, "-->")
	if start < 0 || end < 0 || end <= start+4 {
		return 1
	}
	inner := strings.TrimSpace(s[start+4 : end])
	// inner is "beep" or "beep N"
	parts := strings.Fields(inner)
	if len(parts) >= 2 {
		if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// stripPairs removes matched delimiter pairs from s.
func stripPairs(s, delim string) string {
	for {
		start := strings.Index(s, delim)
		if start < 0 {
			return s
		}
		end := strings.Index(s[start+len(delim):], delim)
		if end < 0 {
			return s
		}
		end += start + len(delim)
		s = s[:start] + s[start+len(delim):end] + s[end+len(delim):]
	}
}
