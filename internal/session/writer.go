package session

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Writer accumulates PTY output and saves it to ~/.shellrelay/sessions/<id>.cast
// Uses a minimal asciinema v2 format for compatibility.
type Writer struct {
	id        string
	startTime time.Time
	file      *os.File
	buf       *bufio.Writer
	written   bool
}

// New creates a session writer. sessionID should be a human-readable timestamp
// (e.g. "2006-01-02_15-04-05"). serverID is included in the asciinema title.
func New(sessionID, serverID string) (*Writer, error) {
	dir, err := sessionsDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, sessionID+".cast")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create session file: %w", err)
	}
	now := time.Now()
	// Human-readable title: "<serverID> · <timestamp>"
	title := serverID + " \u00b7 " + strings.ReplaceAll(sessionID, "_", " ")
	// Write asciinema v2 header
	header := fmt.Sprintf(`{"version":2,"width":220,"height":50,"timestamp":%d,"title":%q}`+"\n",
		now.Unix(), title)
	if _, err := f.WriteString(header); err != nil {
		f.Close()
		return nil, err
	}
	bw := bufio.NewWriterSize(f, 8192)
	return &Writer{id: sessionID, startTime: now, file: f, buf: bw}, nil
}

func (w *Writer) Write(data string) error {
	if w.file == nil {
		return nil
	}
	elapsed := time.Since(w.startTime).Seconds()
	// Escape data for JSON
	escaped := jsonEscape(data)
	line := fmt.Sprintf("[%.6f,\"o\",%s]\n", elapsed, escaped)
	_, err := w.buf.WriteString(line)
	if err == nil {
		w.written = true
	}
	return err
}

func (w *Writer) Close() error {
	if w.file == nil {
		return nil
	}
	if err := w.buf.Flush(); err != nil {
		w.file.Close()
		w.file = nil
		return err
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *Writer) Path() string {
	dir, _ := sessionsDir()
	return filepath.Join(dir, w.id+".cast")
}

func sessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".shellrelay", "sessions")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create sessions dir: %w", err)
	}
	return dir, nil
}

// jsonEscape produces a JSON-encoded string literal (including surrounding quotes)
func jsonEscape(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
