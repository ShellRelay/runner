package relay

// E2E tests for the relay runner against a fake in-process WebSocket server.
//
// Each test spins up an httptest.Server with a gorilla/websocket handler that
// drives a scripted protocol exchange (client_joined → session_start → … →
// session_end).  connect() is called directly so we don't need OS signals;
// the context timeout or server close terminates the loop.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// fakeRelay wraps an httptest.Server that upgrades every connection to WS and
// delegates to a handler provided per-test.
type fakeRelay struct {
	ts  *httptest.Server
	cfg Config
}

func newFakeRelay(t *testing.T, handle func(*websocket.Conn)) *fakeRelay {
	t.Helper()
	fr := &fakeRelay{}
	fr.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("fakeRelay upgrade error: %v", err)
			return
		}
		defer conn.Close()
		handle(conn)
	}))
	t.Cleanup(fr.ts.Close)

	wsURL := "ws://" + strings.TrimPrefix(fr.ts.URL, "http://")
	fr.cfg = Config{
		RelayURL: wsURL,
		ServerID: "test-server",
		Token:    "sr_testtoken",
		Shell:    "/bin/sh",
	}
	return fr
}

// drainUntil reads WebSocket messages from conn until one matching wantType is
// found or deadline is exceeded.  Returns the matching message.
// Signal-only — does not call t.Fatal so it's safe to use from server goroutines.
func drainUntil(conn *websocket.Conn, wantType string, timeout time.Duration) (Msg, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline)
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return Msg{}, false
		}
		var m Msg
		if json.Unmarshal(raw, &m) == nil && m.Type == wantType {
			return m, true
		}
	}
	return Msg{}, false
}

// send writes a JSON message to conn; logs on error (safe from goroutines).
func sendMsg(conn *websocket.Conn, m Msg) {
	conn.WriteJSON(m) //nolint:errcheck
}

// ---------------------------------------------------------------------------
// TestConnWriter_ConcurrentWrites
// Verifies that the mutex-guarded connWriter does not race when many goroutines
// write simultaneously (run with -race to catch data races).
// ---------------------------------------------------------------------------

func TestConnWriter_ConcurrentWrites(t *testing.T) {
	// Server just drains all messages
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	conn, _, err := websocket.DefaultDialer.Dial(
		"ws://"+strings.TrimPrefix(ts.URL, "http://"), nil,
	)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	cw := &connWriter{conn: conn}

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			cw.writeJSON(Msg{Type: "output", Data: fmt.Sprintf("chunk-%d", n)})
		}(i)
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// TestConnect_SessionLifecycle
// Full E2E: connect → client_joined → session_start → end → session_end.
// ---------------------------------------------------------------------------

func TestConnect_SessionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	t.Setenv("HOME", t.TempDir()) // keep sessions files out of real home dir

	sessionStarted := make(chan struct{}, 1)
	sessionEnded := make(chan int, 1)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		sendMsg(conn, Msg{Type: "client_joined"})

		if _, ok := drainUntil(conn, "session_start", 8*time.Second); !ok {
			return // test will time out — no t.Fatal from goroutine
		}
		sessionStarted <- struct{}{}

		sendMsg(conn, Msg{Type: "end"})

		if m, ok := drainUntil(conn, "session_end", 12*time.Second); ok {
			sessionEnded <- m.ExitCode
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connectErr := make(chan error, 1)
	go func() { connectErr <- connect(ctx, fr.cfg) }()

	select {
	case <-sessionStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for session_start")
	}

	select {
	case code := <-sessionEnded:
		t.Logf("session_end received, exit_code=%d", code)
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for session_end")
	}

	// After the handler returns, the server closes the conn → connect returns.
	select {
	case err := <-connectErr:
		t.Logf("connect returned: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("connect did not return after server closed connection")
		cancel()
	}
}

// ---------------------------------------------------------------------------
// TestConnect_BusyWhileSessionActive
// A second client_joined while a session is running must receive "busy".
// ---------------------------------------------------------------------------

func TestConnect_BusyWhileSessionActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	t.Setenv("HOME", t.TempDir())

	gotBusy := make(chan struct{}, 1)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		// Start first session
		sendMsg(conn, Msg{Type: "client_joined"})
		if _, ok := drainUntil(conn, "session_start", 8*time.Second); !ok {
			return
		}

		// While session is running, send second client_joined
		sendMsg(conn, Msg{Type: "client_joined"})

		// Expect busy (may be preceded by output messages from the shell)
		if _, ok := drainUntil(conn, "busy", 5*time.Second); ok {
			gotBusy <- struct{}{}
		}

		// Terminate the first session cleanly
		sendMsg(conn, Msg{Type: "end"})
		drainUntil(conn, "session_end", 10*time.Second) //nolint:errcheck
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() { connect(ctx, fr.cfg) }() //nolint:errcheck

	select {
	case <-gotBusy:
		// pass
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for busy response")
	}
	cancel()
}

// ---------------------------------------------------------------------------
// TestConnect_InputEcho
// Sending input to the PTY produces output containing the echoed text.
// ---------------------------------------------------------------------------

func TestConnect_InputEcho(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	t.Setenv("HOME", t.TempDir())

	const marker = "shellrelay_e2e_marker_12345"
	outputCh := make(chan string, 64)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		sendMsg(conn, Msg{Type: "client_joined"})
		if _, ok := drainUntil(conn, "session_start", 8*time.Second); !ok {
			return
		}

		// Send echo command
		sendMsg(conn, Msg{Type: "input", Data: "echo " + marker + "\n"})

		// Collect output for up to 8 seconds, watching for the marker
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			conn.SetReadDeadline(deadline)
			_, raw, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var m Msg
			if json.Unmarshal(raw, &m) == nil && m.Type == "output" {
				select {
				case outputCh <- m.Data:
				default:
				}
			}
		}

		sendMsg(conn, Msg{Type: "end"})
		drainUntil(conn, "session_end", 5*time.Second) //nolint:errcheck
	})

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	go func() { connect(ctx, fr.cfg) }() //nolint:errcheck

	combined := ""
	timeout := time.After(12 * time.Second)
	for {
		select {
		case chunk := <-outputCh:
			combined += chunk
			if strings.Contains(combined, marker) {
				cancel()
				return // pass
			}
		case <-timeout:
			t.Fatalf("marker %q not found in PTY output; got: %q", marker, combined)
		}
	}
}

// ---------------------------------------------------------------------------
// TestConnect_InvalidShell
// If the configured shell does not exist the runner sends session_end(-1).
// ---------------------------------------------------------------------------

func TestConnect_InvalidShell(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	t.Setenv("HOME", t.TempDir())

	sessionEnded := make(chan int, 1)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		sendMsg(conn, Msg{Type: "client_joined"})
		if m, ok := drainUntil(conn, "session_end", 8*time.Second); ok {
			sessionEnded <- m.ExitCode
		}
	})

	fr.cfg.Shell = "/this/shell/does/not/exist"

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() { connect(ctx, fr.cfg) }() //nolint:errcheck

	select {
	case code := <-sessionEnded:
		if code != -1 {
			t.Errorf("expected exit_code -1 for invalid shell, got %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for session_end with invalid shell")
	}
	cancel()
}

// ---------------------------------------------------------------------------
// TestConnect_ContextCancel
// Cancelling the context makes connect() return promptly.
// ---------------------------------------------------------------------------

func TestConnect_ContextCancel(t *testing.T) {
	connected := make(chan struct{}, 1)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		connected <- struct{}{}
		// Hold the connection open; it will be closed when the runner
		// receives the context cancellation and calls conn.Close().
		conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		conn.ReadMessage() //nolint:errcheck
	})

	ctx, cancel := context.WithCancel(context.Background())

	connectErr := make(chan error, 1)
	go func() { connectErr <- connect(ctx, fr.cfg) }()

	// Wait until runner has connected
	select {
	case <-connected:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not connect within 5 s")
	}

	// Cancel the context
	cancel()

	select {
	case err := <-connectErr:
		t.Logf("connect returned after cancel: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("connect did not return within 5 s after context cancel")
	}
}

// ---------------------------------------------------------------------------
// TestConnect_PendingResize
// A resize message that arrives before client_joined is applied once the PTY
// starts.  We verify the session starts and ends without panicking.
// ---------------------------------------------------------------------------

func TestConnect_PendingResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}
	t.Setenv("HOME", t.TempDir())

	sessionStarted := make(chan struct{}, 1)

	fr := newFakeRelay(t, func(conn *websocket.Conn) {
		// Send resize BEFORE client_joined (stored as pendingWs)
		sendMsg(conn, Msg{Type: "resize", Rows: 30, Cols: 120})

		sendMsg(conn, Msg{Type: "client_joined"})
		if _, ok := drainUntil(conn, "session_start", 8*time.Second); ok {
			sessionStarted <- struct{}{}
		}

		sendMsg(conn, Msg{Type: "end"})
		drainUntil(conn, "session_end", 10*time.Second) //nolint:errcheck
	})

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	go func() { connect(ctx, fr.cfg) }() //nolint:errcheck

	select {
	case <-sessionStarted:
		// pass — pending resize was applied without panic
	case <-time.After(12 * time.Second):
		t.Fatal("timed out waiting for session_start after pending resize")
	}
	cancel()
}
