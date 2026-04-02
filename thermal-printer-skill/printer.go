package main

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	escByte = 0x1B
	gsByte  = 0x1D
	lfByte  = 0x0A

	maxImageDimension = 4096
	printerDotWidth   = 384 // 384 dots for standard 80mm thermal paper
)

// Connection wraps a network/serial connection to the printer.
type Connection struct {
	conn      io.WriteCloser
	closeFunc func() error
}

func (c *Connection) Write(data []byte) (int, error) {
	return c.conn.Write(data)
}

func (c *Connection) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}

// TextStyle defines how text should be formatted.
type TextStyle struct {
	Bold         bool
	DoubleWidth  bool
	DoubleHeight bool
	Centered     bool
	Underline    bool
}

// Printer handles ESC/POS commands for a thermal receipt printer.
type Printer struct {
	conn        *Connection
	paperWidth  int  // characters per line (default 42 for 80mm paper)
	lineSpacing byte // line spacing in printer units (0 = use printer default)
}

// NewPrinter returns a Printer that writes ESC/POS commands to conn.
// Default line spacing is 20 (~2.8mm) for tight, high-density output.
func NewPrinter(conn *Connection) *Printer {
	return &Printer{
		conn:        conn,
		paperWidth:  42,
		lineSpacing: 20,
	}
}

// sanitize strips ESC/POS control characters from user-supplied text.
// Allows printable ASCII/UTF-8 and horizontal tab. Strips ESC, GS, DLE, FS,
// and all other control bytes that could inject printer commands.
func sanitize(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if r >= 0x20 || r == '\t' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Initialize resets the printer and applies configured line spacing.
func (p *Printer) Initialize() error {
	if _, err := p.conn.Write([]byte{escByte, '@'}); err != nil {
		return fmt.Errorf("write init command: %w", err)
	}
	if p.lineSpacing > 0 {
		if err := p.SetLineSpacing(p.lineSpacing); err != nil {
			return fmt.Errorf("set line spacing: %w", err)
		}
	}
	return nil
}

// SetLineSpacing sets the line spacing to n printer units (ESC 3 n).
// Each unit is 1/180 inch on most printers. Default is ~30 (4.2mm).
// Use 20 for tight output (~2.8mm), 16 for very tight (~2.3mm).
func (p *Printer) SetLineSpacing(n byte) error {
	_, err := p.conn.Write([]byte{escByte, '3', n})
	if err != nil {
		return fmt.Errorf("write line spacing: %w", err)
	}
	return nil
}

// ResetLineSpacing restores the printer's default line spacing (ESC 2).
func (p *Printer) ResetLineSpacing() error {
	_, err := p.conn.Write([]byte{escByte, '2'})
	if err != nil {
		return fmt.Errorf("write reset spacing: %w", err)
	}
	return nil
}

// Beep triggers the printer's buzzer (ESC B n t).
// times = number of beeps (1-9), duration = length per beep in ~100ms units (1-9).
func (p *Printer) Beep(times, duration byte) error {
	if times < 1 {
		times = 1
	}
	if times > 9 {
		times = 9
	}
	if duration < 1 {
		duration = 1
	}
	if duration > 9 {
		duration = 9
	}
	_, err := p.conn.Write([]byte{escByte, 'B', times, duration})
	if err != nil {
		return fmt.Errorf("write beep: %w", err)
	}
	return nil
}

// FeedLines advances the paper by n lines (ESC d n).
func (p *Printer) FeedLines(n byte) error {
	_, err := p.conn.Write([]byte{escByte, 'd', n})
	if err != nil {
		return fmt.Errorf("write feed: %w", err)
	}
	return nil
}

// PrintUnderline prints a single line with underline.
func (p *Printer) PrintUnderline(text string) error {
	return p.PrintStyledText(text, TextStyle{Underline: true})
}

// PrintStyledText prints text with the given style, then resets all modes.
func (p *Printer) PrintStyledText(text string, style TextStyle) error {
	// Build the ESC ! mode byte with all applicable bits OR'd together.
	modeByte := byte(0x00)
	if style.Bold {
		modeByte |= 0x08
	}
	if style.DoubleWidth {
		modeByte |= 0x20
	}
	if style.DoubleHeight {
		modeByte |= 0x10
	}

	if _, err := p.conn.Write([]byte{escByte, '!', modeByte}); err != nil {
		return fmt.Errorf("set print mode: %w", err)
	}

	// Use the printer's native centering command so double-width text
	// is centered correctly (manual space-padding breaks with wide fonts).
	if style.Centered {
		if _, err := p.conn.Write([]byte{escByte, 'a', 1}); err != nil {
			return fmt.Errorf("set center alignment: %w", err)
		}
	}

	if style.Underline {
		if _, err := p.conn.Write([]byte{escByte, '-', 1}); err != nil {
			return fmt.Errorf("set underline: %w", err)
		}
	}

	sanitized := sanitize(text)
	if _, err := p.conn.Write([]byte(sanitized)); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	if _, err := p.conn.Write([]byte{lfByte}); err != nil {
		return fmt.Errorf("write line feed: %w", err)
	}

	// Reset every mode that was set, in reverse order.
	if style.Underline {
		if _, err := p.conn.Write([]byte{escByte, '-', 0}); err != nil {
			return fmt.Errorf("reset underline: %w", err)
		}
	}
	if style.Centered {
		if _, err := p.conn.Write([]byte{escByte, 'a', 0}); err != nil {
			return fmt.Errorf("reset alignment: %w", err)
		}
	}
	if _, err := p.conn.Write([]byte{escByte, '!', 0x00}); err != nil {
		return fmt.Errorf("reset print mode: %w", err)
	}

	return nil
}

// PrintBold prints a single line in bold.
func (p *Printer) PrintBold(text string) error {
	return p.PrintStyledText(text, TextStyle{Bold: true})
}

// PrintCentered prints a single line centered.
func (p *Printer) PrintCentered(text string) error {
	return p.PrintStyledText(text, TextStyle{Centered: true})
}

// PrintColumns prints a two-column row padded to the full paper width.
func (p *Printer) PrintColumns(left, right string) error {
	left = sanitize(left)
	right = sanitize(right)

	leftWidth := p.paperWidth/2 - 1
	rightWidth := p.paperWidth - leftWidth

	return p.printRawLine(fitWidth(left, leftWidth) + fitWidth(right, rightWidth))
}

// PrintThreeColumns prints a three-column row padded to the full paper width.
func (p *Printer) PrintThreeColumns(left, center, right string) error {
	left = sanitize(left)
	center = sanitize(center)
	right = sanitize(right)

	colWidth := p.paperWidth / 3

	return p.printRawLine(fitWidth(left, colWidth) + fitWidth(center, colWidth) + fitWidth(right, colWidth))
}

// PrintLine prints a single line of sanitized text followed by a line feed.
func (p *Printer) PrintLine(text string) error {
	return p.printRawLine(sanitize(text))
}

// printRawLine writes already-sanitized text followed by a line feed.
func (p *Printer) printRawLine(text string) error {
	if _, err := p.conn.Write([]byte(text)); err != nil {
		return err
	}
	_, err := p.conn.Write([]byte{lfByte})
	return err
}

// PrintText splits text on newlines and prints each line individually.
func (p *Printer) PrintText(text string) error {
	for _, line := range strings.Split(text, "\n") {
		if err := p.PrintLine(line); err != nil {
			return err
		}
	}
	return nil
}

// PrintSeparator prints a full-width horizontal rule.
func (p *Printer) PrintSeparator() error {
	return p.printRawLine(strings.Repeat("-", p.paperWidth))
}

// PrintQRCode generates a QR code image from data and prints it.
func (p *Printer) PrintQRCode(data string) error {
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("generate QR code: %w", err)
	}
	return p.PrintImage(qr.Image(200))
}

// PrintImage resizes img to the printer's dot width and prints it as a
// monochrome raster bitmap.
func (p *Printer) PrintImage(img image.Image) error {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= 0 || srcH <= 0 {
		return fmt.Errorf("invalid image dimensions: %dx%d", srcW, srcH)
	}
	if srcW > maxImageDimension || srcH > maxImageDimension {
		return fmt.Errorf("image too large: %dx%d (max %d per side)", srcW, srcH, maxImageDimension)
	}

	newH := (printerDotWidth * srcH) / srcW
	if newH <= 0 {
		newH = 1
	}

	dst := nearestNeighborResize(img, printerDotWidth, newH)

	return p.printRasterImage(dst)
}

// printRasterImage converts img to 1-bit monochrome and sends it with GS v 0.
func (p *Printer) printRasterImage(img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	bytesPerLine := (width + 7) / 8

	// GS v 0 m xL xH yL yH d1…dk
	header := []byte{
		gsByte, 'v', 0x30, 0x00, // command + normal density
		byte(bytesPerLine & 0xFF), byte((bytesPerLine >> 8) & 0xFF),
		byte(height & 0xFF), byte((height >> 8) & 0xFF),
	}
	if _, err := p.conn.Write(header); err != nil {
		return fmt.Errorf("write raster header: %w", err)
	}

	lineData := make([]byte, bytesPerLine)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		// Zero the line buffer.
		for i := range lineData {
			lineData[i] = 0
		}
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y).RGBA()
			// Luminance in 0..0xFFFF; dark pixels (< midpoint) print black.
			lum := (299*r + 587*g + 114*b) / 1000
			if lum < 0x8000 {
				lineData[x/8] |= 1 << uint(7-(x%8))
			}
		}
		if _, err := p.conn.Write(lineData); err != nil {
			return fmt.Errorf("write raster line %d: %w", y-bounds.Min.Y, err)
		}
	}

	return nil
}

// PrintImageFromFile decodes an image file and prints it.
// Only regular files are accepted to prevent reading device nodes or pipes.
func (p *Printer) PrintImageFromFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat image file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file: %s", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open image file: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	return p.PrintImage(img)
}

// CutPaper feeds a few lines (so text clears the cutter) then partial-cuts.
func (p *Printer) CutPaper() error {
	// Feed 3 lines so the last printed line is visible above the tear bar.
	if err := p.FeedLines(3); err != nil {
		return fmt.Errorf("pre-cut feed: %w", err)
	}
	_, err := p.conn.Write([]byte{gsByte, 'V', 0})
	if err != nil {
		return fmt.Errorf("write cut command: %w", err)
	}
	return nil
}

// fitWidth pads or truncates text to exactly width characters.
func fitWidth(text string, width int) string {
	runes := []rune(text)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return string(runes) + strings.Repeat(" ", width-len(runes))
}

// nearestNeighborResize scales src to dstW x dstH using nearest-neighbor
// sampling. This is sufficient for thermal printing where the output is
// thresholded to 1-bit monochrome anyway.
func nearestNeighborResize(src image.Image, dstW, dstH int) *image.NRGBA {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, dstW, dstH))

	for y := 0; y < dstH; y++ {
		srcY := srcBounds.Min.Y + (y*srcH)/dstH
		for x := 0; x < dstW; x++ {
			srcX := srcBounds.Min.X + (x*srcW)/dstW
			dst.Set(x, y, src.At(srcX, srcY))
		}
	}
	return dst
}

// PrintTestReceipt prints a receipt that exercises all formatting features.
func (p *Printer) PrintTestReceipt() error {
	if err := p.Initialize(); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	if err := p.PrintStyledText("TEST RECEIPT", TextStyle{Bold: true, DoubleWidth: true, Centered: true}); err != nil {
		return fmt.Errorf("print title: %w", err)
	}
	if err := p.PrintCentered(time.Now().Format("2006-01-02 15:04:05")); err != nil {
		return fmt.Errorf("print timestamp: %w", err)
	}
	if err := p.PrintSeparator(); err != nil {
		return fmt.Errorf("print separator: %w", err)
	}
	if err := p.PrintBold("BOLD TEXT TEST"); err != nil {
		return fmt.Errorf("print bold: %w", err)
	}
	if err := p.PrintColumns("Item", "Price"); err != nil {
		return fmt.Errorf("print column header: %w", err)
	}
	if err := p.PrintColumns("Widget A", "$12.99"); err != nil {
		return fmt.Errorf("print column row: %w", err)
	}
	if err := p.PrintColumns("Gadget B", "$24.50"); err != nil {
		return fmt.Errorf("print column row: %w", err)
	}
	if err := p.PrintSeparator(); err != nil {
		return fmt.Errorf("print separator: %w", err)
	}
	if err := p.PrintCentered("Thank You!"); err != nil {
		return fmt.Errorf("print centered: %w", err)
	}
	if err := p.PrintThreeColumns("Qty", "Item", "Total"); err != nil {
		return fmt.Errorf("print 3-col header: %w", err)
	}
	if err := p.PrintThreeColumns("2", "Widget", "$25.98"); err != nil {
		return fmt.Errorf("print 3-col row: %w", err)
	}
	if err := p.PrintSeparator(); err != nil {
		return fmt.Errorf("print separator: %w", err)
	}

	if err := p.PrintUnderline("Underline test"); err != nil {
		return fmt.Errorf("print underline: %w", err)
	}

	// QR code is best-effort during test; log but don't fail.
	if err := p.PrintQRCode("https://example.com"); err != nil {
		fmt.Fprintf(os.Stderr, "note: QR code skipped: %v\n", err)
	}

	if err := p.PrintCentered("Printer Working!"); err != nil {
		return fmt.Errorf("print footer: %w", err)
	}

	// Beep to signal test complete.
	if err := p.Beep(2, 2); err != nil {
		fmt.Fprintf(os.Stderr, "note: buzzer skipped: %v\n", err)
	}
	if err := p.CutPaper(); err != nil {
		return fmt.Errorf("cut paper: %w", err)
	}

	return nil
}
