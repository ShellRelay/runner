package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setTempHome redirects os.UserHomeDir() to a fresh temp directory for the
// duration of the test so PID/log file helpers don't touch the real ~/.shellrelay.
func setTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	return dir
}

// ---------------------------------------------------------------------------
// pidFile / logFile
// ---------------------------------------------------------------------------

func TestPidFile_Path(t *testing.T) {
	home := setTempHome(t)
	path, err := pidFile()
	if err != nil {
		t.Fatalf("pidFile: %v", err)
	}
	want := filepath.Join(home, ".shellrelay", "shellrelay.pid")
	if path != want {
		t.Errorf("pidFile = %q, want %q", path, want)
	}
}

func TestLogFile_Path(t *testing.T) {
	home := setTempHome(t)
	path, err := logFile()
	if err != nil {
		t.Fatalf("logFile: %v", err)
	}
	want := filepath.Join(home, ".shellrelay", "shellrelay.log")
	if path != want {
		t.Errorf("logFile = %q, want %q", path, want)
	}
}

// ---------------------------------------------------------------------------
// readPID / writePID / removePID
// ---------------------------------------------------------------------------

func TestReadPID_Missing(t *testing.T) {
	setTempHome(t)
	pid, err := readPID()
	if err != nil {
		t.Fatalf("readPID on missing file: %v", err)
	}
	if pid != 0 {
		t.Errorf("expected 0, got %d", pid)
	}
}

func TestWriteReadPID_RoundTrip(t *testing.T) {
	setTempHome(t)
	const want = 42424
	if err := writePID(want); err != nil {
		t.Fatalf("writePID: %v", err)
	}
	got, err := readPID()
	if err != nil {
		t.Fatalf("readPID after write: %v", err)
	}
	if got != want {
		t.Errorf("readPID = %d, want %d", got, want)
	}
}

func TestRemovePID(t *testing.T) {
	setTempHome(t)
	// Write then remove
	if err := writePID(99); err != nil {
		t.Fatalf("writePID: %v", err)
	}
	removePID()
	pid, err := readPID()
	if err != nil {
		t.Fatalf("readPID after remove: %v", err)
	}
	if pid != 0 {
		t.Errorf("expected 0 after removePID, got %d", pid)
	}
}

func TestRemovePID_NoOp_WhenMissing(t *testing.T) {
	setTempHome(t)
	// Should not panic or return an error even when file doesn't exist
	removePID()
}

func TestReadPID_InvalidContent(t *testing.T) {
	home := setTempHome(t)
	// Create the .shellrelay directory and write garbage into the PID file
	dir := filepath.Join(home, ".shellrelay")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "shellrelay.pid")
	if err := os.WriteFile(path, []byte("not-a-number\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readPID()
	if err == nil {
		t.Error("expected error for non-numeric PID file, got nil")
	}
	if !strings.Contains(err.Error(), "invalid PID file") {
		t.Errorf("error message %q should contain 'invalid PID file'", err.Error())
	}
}

func TestReadPID_WhitespaceTrimmed(t *testing.T) {
	home := setTempHome(t)
	dir := filepath.Join(home, ".shellrelay")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	// Write PID with surrounding whitespace — should parse fine
	path := filepath.Join(dir, "shellrelay.pid")
	if err := os.WriteFile(path, []byte("  1234  \n"), 0600); err != nil {
		t.Fatal(err)
	}
	pid, err := readPID()
	if err != nil {
		t.Fatalf("readPID: %v", err)
	}
	if pid != 1234 {
		t.Errorf("expected 1234, got %d", pid)
	}
}

// ---------------------------------------------------------------------------
// isRunning
// ---------------------------------------------------------------------------

func TestIsRunning_Zero(t *testing.T) {
	if isRunning(0) {
		t.Error("isRunning(0) should be false")
	}
}

func TestIsRunning_Negative(t *testing.T) {
	if isRunning(-1) {
		t.Error("isRunning(-1) should be false")
	}
}

func TestIsRunning_CurrentProcess(t *testing.T) {
	if !isRunning(os.Getpid()) {
		t.Errorf("isRunning(%d) should be true (current process)", os.Getpid())
	}
}

func TestIsRunning_NonexistentPID(t *testing.T) {
	// PID 999999999 is extremely unlikely to exist on any system.
	// The test is skipped if it somehow does exist (e.g. PID namespace quirks).
	const fakePID = 999999999
	if isRunning(fakePID) {
		t.Skipf("PID %d unexpectedly exists; skipping", fakePID)
	}
}

// ---------------------------------------------------------------------------
// writePID creates parent directories automatically
// ---------------------------------------------------------------------------

func TestWritePID_CreatesDir(t *testing.T) {
	home := setTempHome(t)
	// Do NOT create ~/.shellrelay beforehand — writePID must create it
	shellrelayDir := filepath.Join(home, ".shellrelay")
	if _, err := os.Stat(shellrelayDir); !os.IsNotExist(err) {
		t.Skip("directory already exists, skipping auto-create test")
	}
	if err := writePID(1); err != nil {
		t.Fatalf("writePID: %v", err)
	}
	if _, err := os.Stat(shellrelayDir); os.IsNotExist(err) {
		t.Error("writePID did not create parent directory")
	}
}
