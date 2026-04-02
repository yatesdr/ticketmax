package main

import (
	"bytes"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParseMarkdown — headings
// ---------------------------------------------------------------------------

func TestParseMarkdown_H1(t *testing.T) {
	elems, err := ParseMarkdown("# Hello World")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	el := elems[0]
	if el.Type != elemTitle {
		t.Errorf("type = %q, want %q", el.Type, elemTitle)
	}
	if el.Text != "Hello World" {
		t.Errorf("text = %q, want %q", el.Text, "Hello World")
	}
	if !el.Style.Bold || !el.Style.DoubleWidth || !el.Style.DoubleHeight || !el.Style.Centered {
		t.Errorf("style = %+v, want bold+dw+dh+centered", el.Style)
	}
}

func TestParseMarkdown_H2(t *testing.T) {
	elems, err := ParseMarkdown("## Subtitle")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	el := elems[0]
	if !el.Style.Bold || !el.Style.DoubleWidth || !el.Style.Centered {
		t.Errorf("H2 style = %+v", el.Style)
	}
	if el.Style.DoubleHeight {
		t.Error("H2 should not have DoubleHeight")
	}
}

func TestParseMarkdown_H3(t *testing.T) {
	elems, err := ParseMarkdown("### Section")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	el := elems[0]
	if !el.Style.Bold {
		t.Error("H3 should be bold")
	}
	if el.Style.DoubleWidth || el.Style.DoubleHeight || el.Style.Centered {
		t.Errorf("H3 should only be bold, got %+v", el.Style)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — separators
// ---------------------------------------------------------------------------

func TestParseMarkdown_Separators(t *testing.T) {
	tests := []string{"---", "***", "___", "- - -", "* * *", "----", "----------"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			elems, err := ParseMarkdown(input)
			if err != nil {
				t.Fatalf("ParseMarkdown(%q): %v", input, err)
			}
			if len(elems) != 1 || elems[0].Type != elemSeparator {
				t.Errorf("expected separator, got %+v", elems)
			}
		})
	}
}

func TestParseMarkdown_NotSeparator(t *testing.T) {
	elems, err := ParseMarkdown("--")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) == 1 && elems[0].Type == elemSeparator {
		t.Error("'--' should not be a separator")
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — bold
// ---------------------------------------------------------------------------

func TestParseMarkdown_BoldLine(t *testing.T) {
	elems, err := ParseMarkdown("**Important notice**")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	if elems[0].Type != elemBold {
		t.Errorf("type = %q, want %q", elems[0].Type, elemBold)
	}
	if elems[0].Text != "Important notice" {
		t.Errorf("text = %q", elems[0].Text)
	}
}

func TestParseMarkdown_InlineBoldStripped(t *testing.T) {
	elems, err := ParseMarkdown("This is **bold** text")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemText {
		t.Fatalf("expected text element, got %+v", elems)
	}
	if elems[0].Text != "This is bold text" {
		t.Errorf("text = %q, want %q", elems[0].Text, "This is bold text")
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — table rows
// ---------------------------------------------------------------------------

func TestParseMarkdown_TwoColumns(t *testing.T) {
	elems, err := ParseMarkdown("| Item | Price |")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 {
		t.Fatalf("expected 1 element, got %d", len(elems))
	}
	el := elems[0]
	if el.Type != elemColumns {
		t.Errorf("type = %q, want %q", el.Type, elemColumns)
	}
	if len(el.Values) != 2 || el.Values[0] != "Item" || el.Values[1] != "Price" {
		t.Errorf("values = %v", el.Values)
	}
}

func TestParseMarkdown_ThreeColumns(t *testing.T) {
	elems, err := ParseMarkdown("| Qty | Item | Total |")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || len(elems[0].Values) != 3 {
		t.Fatalf("expected 3-col row, got %+v", elems)
	}
}

func TestParseMarkdown_TableSeparatorSkipped(t *testing.T) {
	elems, err := ParseMarkdown("| --- | --- |")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 0 {
		t.Errorf("expected 0 elements for table separator, got %d: %+v", len(elems), elems)
	}
}

func TestParseMarkdown_TableAlignmentSeparatorSkipped(t *testing.T) {
	elems, err := ParseMarkdown("| :--- | ---: | :---: |")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 0 {
		t.Errorf("expected 0 elements for alignment separator, got %d", len(elems))
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — image
// ---------------------------------------------------------------------------

func TestParseMarkdown_Image(t *testing.T) {
	elems, err := ParseMarkdown("![logo](assets/logo.png)")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemImage {
		t.Fatalf("expected image, got %+v", elems)
	}
	if elems[0].Path != "assets/logo.png" {
		t.Errorf("path = %q", elems[0].Path)
	}
}

func TestParseMarkdown_ImageMalformed(t *testing.T) {
	elems, err := ParseMarkdown("![broken")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	// Should not produce an image element; falls through to text.
	if len(elems) != 1 || elems[0].Type != elemText {
		t.Errorf("expected text fallback, got %+v", elems)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — QR code
// ---------------------------------------------------------------------------

func TestParseMarkdown_QRCode(t *testing.T) {
	input := "```qr\nhttps://example.com\n```"
	elems, err := ParseMarkdown(input)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemQRCode {
		t.Fatalf("expected qrcode, got %+v", elems)
	}
	if elems[0].Data != "https://example.com" {
		t.Errorf("data = %q", elems[0].Data)
	}
}

func TestParseMarkdown_QRCodeUnclosed(t *testing.T) {
	input := "```qr\nhttps://example.com"
	_, err := ParseMarkdown(input)
	if err == nil {
		t.Error("expected error for unclosed QR block")
	}
}

func TestParseMarkdown_QRCodeMultiline(t *testing.T) {
	input := "```qr\nline1\nline2\n```"
	elems, err := ParseMarkdown(input)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if elems[0].Data != "line1\nline2" {
		t.Errorf("data = %q, want %q", elems[0].Data, "line1\nline2")
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — cut
// ---------------------------------------------------------------------------

func TestParseMarkdown_Cut(t *testing.T) {
	elems, err := ParseMarkdown("<!-- cut -->")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemCut {
		t.Fatalf("expected cut, got %+v", elems)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — feed
// ---------------------------------------------------------------------------

func TestParseMarkdown_EmptyLine(t *testing.T) {
	elems, err := ParseMarkdown("line1\n\nline2")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(elems))
	}
	if elems[1].Type != elemFeed {
		t.Errorf("middle element = %q, want %q", elems[1].Type, elemFeed)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — underline
// ---------------------------------------------------------------------------

func TestParseMarkdown_Underline(t *testing.T) {
	elems, err := ParseMarkdown("<u>Important</u>")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemUnderline {
		t.Fatalf("expected underline, got %+v", elems)
	}
	if elems[0].Text != "Important" {
		t.Errorf("text = %q", elems[0].Text)
	}
}

func TestParseMarkdown_InlineUnderlineStripped(t *testing.T) {
	elems, err := ParseMarkdown("This is <u>underlined</u> text")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemText {
		t.Fatalf("expected text, got %+v", elems)
	}
	if elems[0].Text != "This is underlined text" {
		t.Errorf("text = %q", elems[0].Text)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — beep
// ---------------------------------------------------------------------------

func TestParseMarkdown_Beep(t *testing.T) {
	elems, err := ParseMarkdown("<!-- beep -->")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemBeep {
		t.Fatalf("expected beep, got %+v", elems)
	}
	if elems[0].BeepTimes != 1 {
		t.Errorf("beep times = %d, want 1", elems[0].BeepTimes)
	}
}

func TestParseMarkdown_BeepN(t *testing.T) {
	elems, err := ParseMarkdown("<!-- beep 5 -->")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].BeepTimes != 5 {
		t.Fatalf("expected 5 beeps, got %+v", elems)
	}
}

// ---------------------------------------------------------------------------
// collapseFeeds
// ---------------------------------------------------------------------------

func TestCollapseFeeds_ConsecutiveMerged(t *testing.T) {
	input := []Element{
		{Type: elemText, Text: "a"},
		{Type: elemFeed},
		{Type: elemFeed},
		{Type: elemFeed},
		{Type: elemText, Text: "b"},
	}
	got := collapseFeeds(input)
	// Should be: text, feed, text
	if len(got) != 3 {
		t.Fatalf("expected 3 elements, got %d: %+v", len(got), got)
	}
	if got[1].Type != elemFeed {
		t.Errorf("middle element = %q, want feed", got[1].Type)
	}
}

func TestCollapseFeeds_LeadingStripped(t *testing.T) {
	input := []Element{
		{Type: elemFeed},
		{Type: elemFeed},
		{Type: elemText, Text: "a"},
	}
	got := collapseFeeds(input)
	if len(got) != 1 || got[0].Type != elemText {
		t.Errorf("expected just text, got %+v", got)
	}
}

func TestCollapseFeeds_TrailingStripped(t *testing.T) {
	input := []Element{
		{Type: elemText, Text: "a"},
		{Type: elemFeed},
	}
	got := collapseFeeds(input)
	if len(got) != 1 || got[0].Type != elemText {
		t.Errorf("expected just text, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — regular text
// ---------------------------------------------------------------------------

func TestParseMarkdown_PlainText(t *testing.T) {
	elems, err := ParseMarkdown("Just some text")
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemText {
		t.Fatalf("expected text, got %+v", elems)
	}
	if elems[0].Text != "Just some text" {
		t.Errorf("text = %q", elems[0].Text)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — fenced code block (non-QR)
// ---------------------------------------------------------------------------

func TestParseMarkdown_FencedCodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	elems, err := ParseMarkdown(input)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if len(elems) != 1 || elems[0].Type != elemText {
		t.Fatalf("expected text for code block content, got %+v", elems)
	}
	if elems[0].Text != "fmt.Println(\"hello\")" {
		t.Errorf("text = %q", elems[0].Text)
	}
}

// ---------------------------------------------------------------------------
// ParseMarkdown — full document
// ---------------------------------------------------------------------------

func TestParseMarkdown_FullDocument(t *testing.T) {
	doc := `# Morning Report

## Sales

| Item | Revenue |
| --- | --- |
| Widget A | $1,234 |
| Gadget B | $567 |

---

**Low stock alert**

Widget A: 3 units remaining

` + "```qr" + `
https://dashboard.example.com
` + "```" + `

<!-- cut -->`

	elems, err := ParseMarkdown(doc)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}

	// Verify we get the expected sequence of element types.
	wantTypes := []string{
		elemTitle,     // # Morning Report
		elemFeed,      // blank
		elemTitle,     // ## Sales
		elemFeed,      // blank
		elemColumns,   // | Item | Revenue |
		// | --- | --- | is skipped
		elemColumns,   // | Widget A | $1,234 |
		elemColumns,   // | Gadget B | $567 |
		elemFeed,      // blank
		elemSeparator, // ---
		elemFeed,      // blank
		elemBold,      // **Low stock alert**
		elemFeed,      // blank
		elemText,      // Widget A: 3 units remaining
		elemFeed,      // blank
		elemQRCode,    // ```qr block
		elemFeed,      // blank
		elemCut,       // <!-- cut -->
	}

	if len(elems) != len(wantTypes) {
		t.Fatalf("element count: got %d, want %d", len(elems), len(wantTypes))
	}
	for i, wt := range wantTypes {
		if elems[i].Type != wt {
			t.Errorf("element[%d].Type = %q, want %q", i, elems[i].Type, wt)
		}
	}
}

// ---------------------------------------------------------------------------
// PrintMarkdown — integration (writes to mock)
// ---------------------------------------------------------------------------

func TestPrintMarkdown_Simple(t *testing.T) {
	p, mock := newTestPrinter()

	md := "# Test\n\nHello world\n\n---"
	if err := p.PrintMarkdown(md); err != nil {
		t.Fatalf("PrintMarkdown: %v", err)
	}

	data := mock.buf.Bytes()

	// Should contain the title text.
	if !bytes.Contains(data, []byte("Test")) {
		t.Error("missing title text")
	}
	// Should contain the body text.
	if !bytes.Contains(data, []byte("Hello world")) {
		t.Error("missing body text")
	}
	// Should end with the cut sequence (ESC d 3 + GS V 0).
	cutSeq := []byte{escByte, 'd', 3, gsByte, 'V', 0}
	if !bytes.HasSuffix(data, cutSeq) {
		t.Error("output does not end with cut sequence")
	}
}

func TestPrintMarkdown_NoDuplicateCut(t *testing.T) {
	p, mock := newTestPrinter()

	md := "Hello\n<!-- cut -->"
	if err := p.PrintMarkdown(md); err != nil {
		t.Fatalf("PrintMarkdown: %v", err)
	}

	data := mock.buf.Bytes()
	// CutPaper now sends: ESC d 3 (feed) + GS V 0 (cut)
	// Count occurrences of the full cut sequence.
	cutSeq := []byte{escByte, 'd', 3, gsByte, 'V', 0}
	count := 0
	for i := 0; i <= len(data)-len(cutSeq); i++ {
		if bytes.Equal(data[i:i+len(cutSeq)], cutSeq) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 cut sequence, got %d", count)
	}
}

func TestPrintMarkdown_AutoCut(t *testing.T) {
	p, mock := newTestPrinter()

	md := "Hello world"
	if err := p.PrintMarkdown(md); err != nil {
		t.Fatalf("PrintMarkdown: %v", err)
	}

	// CutPaper sends ESC d 3 + GS V 0
	cutSeq := []byte{escByte, 'd', 3, gsByte, 'V', 0}
	if !bytes.HasSuffix(mock.buf.Bytes(), cutSeq) {
		t.Error("expected auto-cut at end")
	}
}

func TestPrintMarkdown_ErrorPropagation(t *testing.T) {
	ew := &errorWriter{failAfter: 0}
	conn := &Connection{conn: ew, closeFunc: ew.Close}
	p := NewPrinter(conn)

	err := p.PrintMarkdown("# Hello")
	if err == nil {
		t.Error("expected error from failed write")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestIsThematicBreak(t *testing.T) {
	yes := []string{"---", "***", "___", "- - -", "----", "* * * *"}
	no := []string{"--", "**", "text", "-*-", ""}

	for _, s := range yes {
		if !isThematicBreak(s) {
			t.Errorf("isThematicBreak(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isThematicBreak(s) {
			t.Errorf("isThematicBreak(%q) = true, want false", s)
		}
	}
}

func TestStripInlineMarkup(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"**bold**", "bold"},
		{"*italic*", "italic"},
		{"`code`", "code"},
		{"__bold__", "bold"},
		{"_italic_", "italic"},
		{"no markup", "no markup"},
		{"**a** and **b**", "a and b"},
		{"mixed **bold** and *italic*", "mixed bold and italic"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripInlineMarkup(tt.input)
			if got != tt.want {
				t.Errorf("stripInlineMarkup(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTableRow(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   []string
		isNil  bool
	}{
		{"two cols", "| A | B |", []string{"A", "B"}, false},
		{"three cols", "| A | B | C |", []string{"A", "B", "C"}, false},
		{"separator", "| --- | --- |", nil, true},
		{"align sep", "| :--- | ---: |", nil, true},
		{"whitespace", "|  hello  |  world  |", []string{"hello", "world"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTableRow(tt.input)
			if tt.isNil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("col[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseImageLine(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"![logo](logo.png)", "logo.png"},
		{"![](path/to/file.jpg)", "path/to/file.jpg"},
		{"![broken", ""},
		{"![no close](", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseImageLine(tt.input)
			if got != tt.want {
				t.Errorf("parseImageLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsFullLineUnderline(t *testing.T) {
	yes := []string{"<u>hello</u>", "<u>two words</u>"}
	no := []string{"<u>", "</u>", "<u></u>", "not underlined", "<u>a</u> <u>b</u>"}

	for _, s := range yes {
		if !isFullLineUnderline(s) {
			t.Errorf("isFullLineUnderline(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isFullLineUnderline(s) {
			t.Errorf("isFullLineUnderline(%q) = true, want false", s)
		}
	}
}

func TestParseBeepDirective(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"<!-- beep -->", 1},
		{"<!-- beep 3 -->", 3},
		{"<!-- beep 9 -->", 9},
		{"<!-- beep abc -->", 1},
	}
	for _, tt := range tests {
		got := parseBeepDirective(tt.input)
		if got != tt.want {
			t.Errorf("parseBeepDirective(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestStripHTMLTag(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"<u>text</u>", "text"},
		{"a <u>b</u> c", "a b c"},
		{"no tags", "no tags"},
		{"<u>a</u> and <u>b</u>", "a and b"},
	}
	for _, tt := range tests {
		got := stripHTMLTag(tt.input, "u")
		if got != tt.want {
			t.Errorf("stripHTMLTag(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsFullLineBold(t *testing.T) {
	yes := []string{"**hello**", "**two words**"}
	no := []string{"**", "****", "**a** b **c**", "not bold", "*single*"}

	for _, s := range yes {
		if !isFullLineBold(s) {
			t.Errorf("isFullLineBold(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isFullLineBold(s) {
			t.Errorf("isFullLineBold(%q) = true, want false", s)
		}
	}
}

func TestStripPairs(t *testing.T) {
	tests := []struct {
		input, delim, want string
	}{
		{"**bold**", "**", "bold"},
		{"no delim", "**", "no delim"},
		{"**a** and **b**", "**", "a and b"},
		{"*single*", "*", "single"},
		{"unmatched *star", "*", "unmatched *star"},
	}
	for _, tt := range tests {
		name := tt.input + "/" + tt.delim
		t.Run(name, func(t *testing.T) {
			got := stripPairs(tt.input, tt.delim)
			if got != tt.want {
				t.Errorf("stripPairs(%q, %q) = %q, want %q", tt.input, tt.delim, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// envOrDefault (moved test here since it's in main.go)
// ---------------------------------------------------------------------------

func TestEnvOrDefault(t *testing.T) {
	got := envOrDefault("THERMAL_PRINTER_TEST_UNSET_XYZ", "fallback")
	if got != "fallback" {
		t.Errorf("expected fallback, got %q", got)
	}

	t.Setenv("THERMAL_PRINTER_TEST_SET", "custom")
	got = envOrDefault("THERMAL_PRINTER_TEST_SET", "fallback")
	if got != "custom" {
		t.Errorf("expected 'custom', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// readInput
// ---------------------------------------------------------------------------

func TestReadInput_TooManyArgs(t *testing.T) {
	_, err := readInput([]string{"a", "b"})
	if err == nil {
		t.Error("expected error for too many args")
	}
	if !strings.Contains(err.Error(), "expected one file argument") {
		t.Errorf("unexpected error: %v", err)
	}
}
