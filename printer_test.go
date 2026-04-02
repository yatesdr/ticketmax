package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestPrinter() (*Printer, *bytes.Buffer) {
	var buf bytes.Buffer
	return NewPrinter(&buf), &buf
}

// errorWriter fails after failAfter bytes have been written.
type errorWriter struct {
	failAfter int
	written   int
}

func (e *errorWriter) Write(p []byte) (int, error) {
	if e.written+len(p) > e.failAfter {
		return 0, fmt.Errorf("simulated write error")
	}
	e.written += len(p)
	return len(p), nil
}

// ---------------------------------------------------------------------------
// sanitize
// ---------------------------------------------------------------------------

func TestSanitize(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"plain text", "Hello World", "Hello World"},
		{"tab preserved", "col1\tcol2", "col1\tcol2"},
		{"ESC stripped", "before\x1bafter", "beforeafter"},
		{"GS stripped", "before\x1dafter", "beforeafter"},
		{"DLE stripped", "before\x10after", "beforeafter"},
		{"FS stripped", "before\x1cafter", "beforeafter"},
		{"LF stripped", "before\nafter", "beforeafter"},
		{"CR stripped", "before\rafter", "beforeafter"},
		{"NUL stripped", "before\x00after", "beforeafter"},
		{"mixed control", "\x1b@\x1dV\x00test", "@Vtest"},
		{"empty", "", ""},
		{"all control", "\x00\x01\x1b\x1d", ""},
		{"unicode", "Hello 世界", "Hello 世界"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitize(tt.input); got != tt.want {
				t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// fitWidth
// ---------------------------------------------------------------------------

func TestFitWidth(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{"exact", "hello", 5, "hello"},
		{"pad", "hi", 5, "hi   "},
		{"truncate", "hello world", 5, "hello"},
		{"empty", "", 3, "   "},
		{"zero width", "hello", 0, ""},
		{"unicode pad", "日本", 5, "日本   "},
		{"unicode truncate", "日本語テスト", 3, "日本語"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fitWidth(tt.text, tt.width); got != tt.want {
				t.Errorf("fitWidth(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Initialize / CutPaper
// ---------------------------------------------------------------------------

func TestInitialize(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	data := mock.Bytes()
	// ESC @ (init) + ESC 3 20 (line spacing)
	want := []byte{escByte, '@', escByte, '3', 20}
	if !bytes.Equal(data, want) {
		t.Errorf("got %v, want %v", data, want)
	}
}

func TestInitialize_NoSpacing(t *testing.T) {
	p, mock := newTestPrinter()
	p.lineSpacing = 0 // disable custom spacing
	if err := p.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	want := []byte{escByte, '@'}
	if !bytes.Equal(mock.Bytes(), want) {
		t.Errorf("got %v, want %v", mock.Bytes(), want)
	}
}

func TestCutPaper(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.CutPaper(); err != nil {
		t.Fatalf("CutPaper: %v", err)
	}
	data := mock.Bytes()
	// ESC d 3 (feed 3 lines) + GS V 1 (partial cut)
	want := []byte{escByte, 'd', 3, gsByte, 'V', 1}
	if !bytes.Equal(data, want) {
		t.Errorf("got %v, want %v", data, want)
	}
}

// ---------------------------------------------------------------------------
// PrintStyledText — bold
// ---------------------------------------------------------------------------

func TestPrintStyledText_Bold(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintStyledText("TEST", TextStyle{Bold: true}); err != nil {
		t.Fatalf("PrintStyledText: %v", err)
	}
	data := mock.Bytes()

	// Must contain ESC ! 0x08 (bold bit set)
	if !bytes.Contains(data, []byte{escByte, '!', 0x08}) {
		t.Errorf("missing bold mode command (ESC ! 0x08) in %v", data)
	}
	// Must contain the text
	if !bytes.Contains(data, []byte("TEST")) {
		t.Error("text 'TEST' not found in output")
	}
	// Must reset mode
	if !bytes.Contains(data, []byte{escByte, '!', 0x00}) {
		t.Error("missing mode reset (ESC ! 0x00)")
	}
}

func TestPrintStyledText_BoldDoubleWidth(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintStyledText("X", TextStyle{Bold: true, DoubleWidth: true}); err != nil {
		t.Fatalf("PrintStyledText: %v", err)
	}
	// bold=0x08 | double-width=0x20 => 0x28
	if !bytes.Contains(mock.Bytes(), []byte{escByte, '!', 0x28}) {
		t.Errorf("expected ESC ! 0x28 for bold+double-width, got %v", mock.Bytes())
	}
}

func TestPrintStyledText_AllModes(t *testing.T) {
	p, mock := newTestPrinter()
	style := TextStyle{Bold: true, DoubleWidth: true, DoubleHeight: true}
	if err := p.PrintStyledText("X", style); err != nil {
		t.Fatalf("PrintStyledText: %v", err)
	}
	// bold=0x08 | double-height=0x10 | double-width=0x20 => 0x38
	if !bytes.Contains(mock.Bytes(), []byte{escByte, '!', 0x38}) {
		t.Errorf("expected ESC ! 0x38, got %v", mock.Bytes())
	}
}

// ---------------------------------------------------------------------------
// PrintStyledText — centering via ESC a
// ---------------------------------------------------------------------------

func TestPrintStyledText_Centered(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintStyledText("CENTER", TextStyle{Centered: true}); err != nil {
		t.Fatalf("PrintStyledText: %v", err)
	}
	data := mock.Bytes()

	if !bytes.Contains(data, []byte{escByte, 'a', 1}) {
		t.Error("missing ESC a 1 (center)")
	}
	if !bytes.Contains(data, []byte{escByte, 'a', 0}) {
		t.Error("missing ESC a 0 (reset to left)")
	}
}

// ---------------------------------------------------------------------------
// PrintStyledText — underline
// ---------------------------------------------------------------------------

func TestPrintStyledText_Underline(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintStyledText("UL", TextStyle{Underline: true}); err != nil {
		t.Fatalf("PrintStyledText: %v", err)
	}
	data := mock.Bytes()

	if !bytes.Contains(data, []byte{escByte, '-', 1}) {
		t.Error("missing ESC - 1 (underline on)")
	}
	if !bytes.Contains(data, []byte{escByte, '-', 0}) {
		t.Error("missing ESC - 0 (underline off)")
	}
}

// ---------------------------------------------------------------------------
// PrintLine sanitizes
// ---------------------------------------------------------------------------

func TestPrintLine_Sanitizes(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintLine("Hello\x1b@World"); err != nil {
		t.Fatalf("PrintLine: %v", err)
	}
	out := mock.String()
	if strings.Contains(out, "\x1b") {
		t.Error("ESC byte leaked through sanitize")
	}
	if !strings.Contains(out, "Hello@World") {
		t.Errorf("expected sanitized text, got %q", out)
	}
}

// ---------------------------------------------------------------------------
// PrintColumns
// ---------------------------------------------------------------------------

func TestPrintColumns_Width(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintColumns("Item", "Price"); err != nil {
		t.Fatalf("PrintColumns: %v", err)
	}
	out := mock.String()
	// Line is paper width (42) + LF (1 byte)
	if len(out) != 43 {
		t.Errorf("expected line length 43 (42 + LF), got %d", len(out))
	}
	if out[len(out)-1] != lfByte {
		t.Error("line does not end with LF")
	}
}

func TestPrintColumns_Sanitizes(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintColumns("A\x1b@", "B\x1d"); err != nil {
		t.Fatalf("PrintColumns: %v", err)
	}
	out := mock.Bytes()
	if bytes.ContainsAny(out, "\x1b\x1d") {
		t.Error("control characters leaked through column sanitization")
	}
}

// ---------------------------------------------------------------------------
// PrintThreeColumns
// ---------------------------------------------------------------------------

func TestPrintThreeColumns(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintThreeColumns("Qty", "Item", "Total"); err != nil {
		t.Fatalf("PrintThreeColumns: %v", err)
	}
	out := mock.String()
	if !strings.Contains(out, "Qty") || !strings.Contains(out, "Item") || !strings.Contains(out, "Total") {
		t.Errorf("missing column values in %q", out)
	}
}

// ---------------------------------------------------------------------------
// PrintSeparator
// ---------------------------------------------------------------------------

func TestPrintSeparator(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintSeparator(); err != nil {
		t.Fatalf("PrintSeparator: %v", err)
	}
	want := strings.Repeat("-", 42) + string([]byte{lfByte})
	if mock.String() != want {
		t.Errorf("got %q, want %q", mock.String(), want)
	}
}

// ---------------------------------------------------------------------------
// Error propagation
// ---------------------------------------------------------------------------

func TestPrintLine_PropagatesWriteError(t *testing.T) {
	ew := &errorWriter{failAfter: 0}
	p := NewPrinter(ew)

	if err := p.PrintLine("test"); err == nil {
		t.Error("expected error from failed write, got nil")
	}
}

func TestInitialize_PropagatesWriteError(t *testing.T) {
	ew := &errorWriter{failAfter: 0}
	p := NewPrinter(ew)

	if err := p.Initialize(); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCutPaper_PropagatesWriteError(t *testing.T) {
	ew := &errorWriter{failAfter: 0}
	p := NewPrinter(ew)

	if err := p.CutPaper(); err == nil {
		t.Error("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// PrintImage — dimensions
// ---------------------------------------------------------------------------

func TestPrintImage_InvalidDimensions(t *testing.T) {
	p, _ := newTestPrinter()
	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	if err := p.PrintImage(img); err == nil {
		t.Error("expected error for zero-dimension image")
	}
}

func TestPrintImage_TooLarge(t *testing.T) {
	p, _ := newTestPrinter()
	img := image.NewRGBA(image.Rect(0, 0, maxImageDimension+1, 100))
	if err := p.PrintImage(img); err == nil {
		t.Error("expected error for oversized image")
	}
}

func TestPrintImage_SmallBlackSquare(t *testing.T) {
	p, mock := newTestPrinter()

	// 8x8 black image → all pixels should print as 1-bits
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.Black)
		}
	}

	if err := p.PrintImage(img); err != nil {
		t.Fatalf("PrintImage: %v", err)
	}

	data := mock.Bytes()
	// Output: ESC a 1 (center) + GS v 0 header + bitmap + ESC a 0 (reset)
	// First 3 bytes are the center alignment command.
	if len(data) < 11 {
		t.Fatalf("output too short: %d bytes", len(data))
	}
	if !bytes.Equal(data[:3], []byte{escByte, 'a', 1}) {
		t.Errorf("expected center alignment prefix, got %v", data[:3])
	}
	// Raster header starts at offset 3.
	if data[3] != gsByte || data[4] != 'v' || data[5] != 0x30 {
		t.Errorf("unexpected raster header: %v", data[3:7])
	}
	// Last 3 bytes reset alignment.
	tail := data[len(data)-3:]
	if !bytes.Equal(tail, []byte{escByte, 'a', 0}) {
		t.Errorf("expected alignment reset suffix, got %v", tail)
	}
}

func TestPrintImage_WhiteSquare(t *testing.T) {
	p, mock := newTestPrinter()

	// 8x8 white image → all pixels should be 0-bits
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.White)
		}
	}

	if err := p.PrintImage(img); err != nil {
		t.Fatalf("PrintImage: %v", err)
	}

	data := mock.Bytes()
	// Skip 3 (center align) + 8 (raster header) = 11 bytes prefix,
	// and strip 3 (alignment reset) bytes from the end.
	bitmap := data[11 : len(data)-3]
	for i, b := range bitmap {
		if b != 0x00 {
			t.Errorf("byte %d = 0x%02x, want 0x00 for white image", i, b)
			break
		}
	}
}

// ---------------------------------------------------------------------------
// PrintImageFromFile — rejects non-regular files
// ---------------------------------------------------------------------------

func TestPrintImageFromFile_RejectsDirectory(t *testing.T) {
	p, _ := newTestPrinter()
	if err := p.PrintImageFromFile(t.TempDir()); err == nil {
		t.Error("expected error when given a directory")
	}
}

func TestPrintImageFromFile_RejectsNonexistent(t *testing.T) {
	p, _ := newTestPrinter()
	if err := p.PrintImageFromFile("/nonexistent/path.png"); err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPrintImageFromFile_DecodesRealPNG(t *testing.T) {
	p, mock := newTestPrinter()

	// Create a small valid PNG in a temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.Black)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode PNG: %v", err)
	}
	f.Close()

	if err := p.PrintImageFromFile(path); err != nil {
		t.Fatalf("PrintImageFromFile: %v", err)
	}
	if mock.Len() == 0 {
		t.Error("expected output, got nothing")
	}
}

// ---------------------------------------------------------------------------
// Beep
// ---------------------------------------------------------------------------

func TestBeep(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.Beep(3, 2); err != nil {
		t.Fatalf("Beep: %v", err)
	}
	want := []byte{escByte, 'B', 3, 2}
	if !bytes.Equal(mock.Bytes(), want) {
		t.Errorf("got %v, want %v", mock.Bytes(), want)
	}
}

func TestBeep_Clamps(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.Beep(0, 99); err != nil {
		t.Fatalf("Beep: %v", err)
	}
	data := mock.Bytes()
	// times clamped to 1, duration clamped to 9
	if data[2] != 1 || data[3] != 9 {
		t.Errorf("expected clamped values [1,9], got [%d,%d]", data[2], data[3])
	}
}

func TestBeep_PropagatesError(t *testing.T) {
	ew := &errorWriter{failAfter: 0}
	p := NewPrinter(ew)
	if err := p.Beep(1, 1); err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// SetLineSpacing / FeedLines / PrintUnderline
// ---------------------------------------------------------------------------

func TestSetLineSpacing(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.SetLineSpacing(24); err != nil {
		t.Fatalf("SetLineSpacing: %v", err)
	}
	want := []byte{escByte, '3', 24}
	if !bytes.Equal(mock.Bytes(), want) {
		t.Errorf("got %v, want %v", mock.Bytes(), want)
	}
}

func TestFeedLines(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.FeedLines(5); err != nil {
		t.Fatalf("FeedLines: %v", err)
	}
	want := []byte{escByte, 'd', 5}
	if !bytes.Equal(mock.Bytes(), want) {
		t.Errorf("got %v, want %v", mock.Bytes(), want)
	}
}

func TestPrintUnderline(t *testing.T) {
	p, mock := newTestPrinter()
	if err := p.PrintUnderline("test"); err != nil {
		t.Fatalf("PrintUnderline: %v", err)
	}
	data := mock.Bytes()
	if !bytes.Contains(data, []byte{escByte, '-', 1}) {
		t.Error("missing underline on")
	}
	if !bytes.Contains(data, []byte{escByte, '-', 0}) {
		t.Error("missing underline off")
	}
	if !bytes.Contains(data, []byte("test")) {
		t.Error("missing text")
	}
}

// ---------------------------------------------------------------------------
// NewPrinter defaults
// ---------------------------------------------------------------------------

func TestNewPrinter_Defaults(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	if p.paperWidth != 42 {
		t.Errorf("default paper width = %d, want 42", p.paperWidth)
	}
	if p.lineSpacing != 20 {
		t.Errorf("default line spacing = %d, want 20", p.lineSpacing)
	}
}

