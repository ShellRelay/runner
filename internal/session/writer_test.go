package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helpers -----------------------------------------------------------------

// withHome overrides HOME so sessionsDir() resolves to a temp directory.
func withHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
}

func TestClose_FlushError(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("test-flush-err", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	// Buffer some data then close the underlying file so Flush() fails.
	if err := w.Write("buffered data"); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	w.file.Close() // close underlying fd — Flush() will now fail

	if err := w.Close(); err == nil {
		t.Error("expected error from Close() when underlying file is closed, got nil")
	}
}

// -------------------------------------------------------------------------
// sessionsDir / New error paths (requires non-root execution)
// -------------------------------------------------------------------------

func TestSessionsDir_MkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — chmod has no effect")
	}
	tmp := t.TempDir()
	withHome(t, tmp)

	// Create ~/.shellrelay but make it read-only so sessions/ sub-dir can't be created.
	shellrelayDir := filepath.Join(tmp, ".shellrelay")
	if err := os.MkdirAll(shellrelayDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shellrelayDir, 0500); err != nil {
		t.Skip("cannot change dir permissions")
	}
	t.Cleanup(func() { os.Chmod(shellrelayDir, 0700) }) //nolint:errcheck

	_, err := sessionsDir()
	if err == nil {
		t.Error("expected error when parent dir is read-only, got nil")
	}
}

func TestNew_CreateFileError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root — chmod has no effect")
	}
	tmp := t.TempDir()
	withHome(t, tmp)

	// Create sessions dir then make it read-only so os.Create fails.
	sessDir := filepath.Join(tmp, ".shellrelay", "sessions")
	if err := os.MkdirAll(sessDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sessDir, 0500); err != nil {
		t.Skip("cannot change dir permissions")
	}
	t.Cleanup(func() { os.Chmod(sessDir, 0700) }) //nolint:errcheck

	_, err := New("test-readonly", "srv")
	if err == nil {
		t.Error("expected error when sessions dir is read-only, got nil")
	}
}

// -------------------------------------------------------------------------
// jsonEscape
// -------------------------------------------------------------------------

func TestJsonEscape(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "hello", want: `"hello"`},
		{name: "double quote", input: `say "hi"`, want: `"say \"hi\""`},
		{name: "backslash", input: `a\b`, want: `"a\\b"`},
		{name: "newline", input: "line1\nline2", want: `"line1\nline2"`},
		{name: "carriage return", input: "a\rb", want: `"a\rb"`},
		{name: "tab", input: "a\tb", want: `"a\tb"`},
		{name: "control char", input: "a\x01b", want: `"a\u0001b"`},
		{name: "empty string", input: "", want: `""`},
		{name: "unicode", input: "hello 世界", want: `"hello 世界"`},
		{
			name:  "mixed special chars",
			input: "line1\nline2\t\"quoted\"\r\n",
			want:  `"line1\nline2\t\"quoted\"\r\n"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonEscape(tc.input)
			if got != tc.want {
				t.Errorf("jsonEscape(%q) = %s, want %s", tc.input, got, tc.want)
			}
		})
	}
}

// Verify that the output of jsonEscape is always valid JSON.
func TestJsonEscape_ValidJSON(t *testing.T) {
	inputs := []string{
		"",
		"simple",
		"with\nnewline",
		`with"quote`,
		"with\ttab",
		"null\x00byte",
		"\x1b[31mred\x1b[0m", // ANSI escape
	}
	for _, input := range inputs {
		escaped := jsonEscape(input)
		var decoded string
		if err := json.Unmarshal([]byte(escaped), &decoded); err != nil {
			t.Errorf("jsonEscape(%q) = %s  — not valid JSON: %v", input, escaped, err)
		}
	}
}

// -------------------------------------------------------------------------
// New — file creation and header
// -------------------------------------------------------------------------

func TestNew_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("2024-01-15_10-30-00", "srv-1")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer w.Close()

	path := filepath.Join(tmp, ".shellrelay", "sessions", "2024-01-15_10-30-00.cast")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file %s to exist", path)
	}
}

func TestNew_HeaderIsValidJSON(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("2024-01-15_10-30-00", "srv-test")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	w.Close()

	data, err := os.ReadFile(filepath.Join(tmp, ".shellrelay", "sessions", "2024-01-15_10-30-00.cast"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 1 {
		t.Fatal("cast file is empty")
	}

	var hdr struct {
		Version   int    `json:"version"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Timestamp int64  `json:"timestamp"`
		Title     string `json:"title"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &hdr); err != nil {
		t.Fatalf("header is not valid JSON: %v\nheader: %s", err, lines[0])
	}
	if hdr.Version != 2 {
		t.Errorf("version = %d, want 2", hdr.Version)
	}
	if hdr.Timestamp <= 0 {
		t.Error("timestamp should be > 0")
	}
	if !strings.Contains(hdr.Title, "srv-test") {
		t.Errorf("title = %q, expected to contain %q", hdr.Title, "srv-test")
	}
}

// -------------------------------------------------------------------------
// Write — produces valid asciinema v2 event lines
// -------------------------------------------------------------------------

func TestWrite_ProducesValidEventLines(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("test-write", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if err := w.Write("hello world"); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if err := w.Write("line two\n"); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	w.Close()

	data, err := os.ReadFile(filepath.Join(tmp, ".shellrelay", "sessions", "test-write.cast"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	// line 0 = header, line 1 = first event, line 2 = second event
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}

	for i := 1; i < len(lines); i++ {
		var event [3]json.RawMessage
		if err := json.Unmarshal([]byte(lines[i]), &event); err != nil {
			t.Errorf("event line %d is not valid JSON array: %v\nline: %s", i, err, lines[i])
			continue
		}
		// Check timestamp is a number
		var ts float64
		if err := json.Unmarshal(event[0], &ts); err != nil {
			t.Errorf("event[0] is not a number: %v", err)
		}
		// Check event type is "o"
		var evType string
		if err := json.Unmarshal(event[1], &evType); err != nil {
			t.Errorf("event[1] is not a string: %v", err)
		}
		if evType != "o" {
			t.Errorf("event type = %q, want %q", evType, "o")
		}
		// Check data is a valid JSON string
		var evData string
		if err := json.Unmarshal(event[2], &evData); err != nil {
			t.Errorf("event[2] is not a valid JSON string: %v", err)
		}
	}
}

func TestWrite_SpecialCharacters(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("test-special", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Write data with special chars that need JSON escaping
	specialData := "\x1b[31mred\x1b[0m \"quoted\" back\\slash\ttab\nnewline"
	if err := w.Write(specialData); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	w.Close()

	data, err := os.ReadFile(filepath.Join(tmp, ".shellrelay", "sessions", "test-special.cast"))
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	// Parse the event and verify the data round-trips correctly
	var event [3]json.RawMessage
	if err := json.Unmarshal([]byte(lines[1]), &event); err != nil {
		t.Fatalf("event line is not valid JSON: %v\nline: %s", err, lines[1])
	}
	var decoded string
	if err := json.Unmarshal(event[2], &decoded); err != nil {
		t.Fatalf("event data is not valid JSON string: %v", err)
	}
	if decoded != specialData {
		t.Errorf("round-trip mismatch:\n  got  %q\n  want %q", decoded, specialData)
	}
}

// -------------------------------------------------------------------------
// Close
// -------------------------------------------------------------------------

func TestClose_NilWriter(t *testing.T) {
	w := &Writer{} // file is nil
	if err := w.Close(); err != nil {
		t.Errorf("Close() on nil-file writer should not error, got: %v", err)
	}
}

func TestClose_DoubleClose(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("test-dblclose", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}
	// Second close should be a no-op (file is nil).
	if err := w.Close(); err != nil {
		t.Errorf("second Close() should not error, got: %v", err)
	}
}

// -------------------------------------------------------------------------
// Write on closed writer
// -------------------------------------------------------------------------

func TestWrite_AfterClose(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("test-closed-write", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	w.Close()

	// Write after close should be a no-op (file is nil), not panic.
	if err := w.Write("data"); err != nil {
		t.Errorf("Write() after Close() should be nil, got: %v", err)
	}
}

// -------------------------------------------------------------------------
// Path
// -------------------------------------------------------------------------

func TestWriter_Path(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	w, err := New("myid", "srv")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer w.Close()

	got := w.Path()
	want := filepath.Join(tmp, ".shellrelay", "sessions", "myid.cast")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}
