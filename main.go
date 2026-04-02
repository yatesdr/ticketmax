package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	addr := flag.String("addr", envOrDefault("PRINTER_ADDR", "127.0.0.1:9100"),
		"Printer address as host:port (env: PRINTER_ADDR)")
	timeout := flag.Duration("timeout", 5*time.Second,
		"Network connection timeout")
	width := flag.Int("width", 42,
		"Paper width in characters (1-120)")
	spacing := flag.Int("spacing", 20,
		"Line spacing in printer units (0=printer default, 16=very tight, 20=tight, 30=normal)")
	test := flag.Bool("test", false,
		"Print a test receipt and exit")
	status := flag.Bool("status", false,
		"Check printer connectivity and exit")
	version := flag.Bool("version", false,
		"Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: thermal-printer [flags] <file.md | ->\n\n")
		fmt.Fprintf(os.Stderr, "Print a markdown file to an ESC/POS thermal receipt printer.\n")
		fmt.Fprintf(os.Stderr, "Use - to read from stdin.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nMarkdown support:\n")
		fmt.Fprintf(os.Stderr, "  # Heading           bold, double-size, centered\n")
		fmt.Fprintf(os.Stderr, "  ## Heading           bold, double-width, centered\n")
		fmt.Fprintf(os.Stderr, "  ### Heading          bold\n")
		fmt.Fprintf(os.Stderr, "  **bold text**        bold line\n")
		fmt.Fprintf(os.Stderr, "  <u>text</u>          underlined line\n")
		fmt.Fprintf(os.Stderr, "  ---                  separator\n")
		fmt.Fprintf(os.Stderr, "  | A | B |            two columns\n")
		fmt.Fprintf(os.Stderr, "  | A | B | C |        three columns\n")
		fmt.Fprintf(os.Stderr, "  ![alt](path)         image\n")
		fmt.Fprintf(os.Stderr, "  ```qr ... ```        QR code\n")
		fmt.Fprintf(os.Stderr, "  <!-- cut -->          cut paper\n")
		fmt.Fprintf(os.Stderr, "  <!-- beep -->         buzzer (1 beep)\n")
		fmt.Fprintf(os.Stderr, "  <!-- beep N -->       buzzer (N beeps)\n")
	}

	flag.Parse()

	if *version {
		fmt.Println("thermal-printer " + Version)
		return
	}

	if *width < 1 || *width > 120 {
		log.Fatalf("invalid width %d: must be between 1 and 120", *width)
	}
	if *spacing < 0 || *spacing > 255 {
		log.Fatalf("invalid spacing %d: must be between 0 and 255", *spacing)
	}

	// --status: check connectivity and exit.
	if *status {
		if err := checkStatus(*addr, *timeout); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("OK")
		return
	}

	// Connect to printer.
	conn, err := net.DialTimeout("tcp", *addr, *timeout)
	if err != nil {
		log.Fatalf("connect: dial %s: %v", *addr, err)
	}
	defer conn.Close()

	printer := NewPrinter(conn)
	printer.paperWidth = *width
	printer.lineSpacing = byte(*spacing)

	// --test: print test receipt and exit.
	if *test {
		if err := printer.PrintTestReceipt(); err != nil {
			log.Fatalf("test receipt: %v", err)
		}
		fmt.Println("Test receipt printed successfully")
		return
	}

	// Read markdown from file argument or stdin.
	md, err := readInput(flag.Args())
	if err != nil {
		log.Fatalf("read input: %v", err)
	}

	if err := printer.PrintMarkdown(md); err != nil {
		log.Fatalf("print: %v", err)
	}

	fmt.Println("Printed successfully")
}

// readInput returns the markdown content from a file path, stdin ("-"), or
// defaults to stdin if no arguments are given.
func readInput(args []string) (string, error) {
	var r io.Reader

	switch {
	case len(args) == 0 || (len(args) == 1 && args[0] == "-"):
		// Check if stdin is a terminal (no piped input).
		info, err := os.Stdin.Stat()
		if err != nil {
			return "", fmt.Errorf("stat stdin: %w", err)
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", fmt.Errorf("no input: provide a markdown file or pipe to stdin")
		}
		r = os.Stdin
	case len(args) == 1:
		path := args[0]
		f, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open %q: %w", path, err)
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return "", fmt.Errorf("stat %q: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("not a regular file: %s", path)
		}
		r = f
	default:
		return "", fmt.Errorf("expected one file argument, got %d", len(args))
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return string(data), nil
}

func checkStatus(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn.Close()
}
