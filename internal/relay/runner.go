package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ShellRelay/runner/internal/session"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// Message types exchanged with the relay server
type Msg struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	Rows     uint16 `json:"rows,omitempty"`
	Cols     uint16 `json:"cols,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// Config holds runner connection parameters
type Config struct {
	RelayURL string // wss://... base URL (no path)
	ServerID string
	Token    string
	Shell    string // e.g. /bin/zsh
}

// connWriter provides goroutine-safe writes to a websocket connection.
type connWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (cw *connWriter) writeJSON(v interface{}) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.conn.WriteJSON(v)
}

func (cw *connWriter) writeControl(messageType int, data []byte, deadline time.Time) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.conn.WriteControl(messageType, data, deadline)
}

// Run connects to the relay and manages the PTY lifecycle.
// Reconnects automatically with exponential backoff.
func Run(cfg Config) {
	// Validate relay URL scheme
	if !strings.HasPrefix(cfg.RelayURL, "wss://") && !strings.HasPrefix(cfg.RelayURL, "ws://") {
		log.Fatalf("[runner] relay URL must start with wss:// or ws://, got: %s", cfg.RelayURL)
	}

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// quit is closed once when a signal is received. Unlike sigCh it can be
	// selected on multiple times without being consumed, which prevents the
	// signal-drain race where the inner goroutine consumes the signal before
	// the outer reconnect loop can observe it.
	quit := make(chan struct{})
	go func() {
		sig := <-sigCh
		log.Printf("[runner] received %s, shutting down...", sig)
		close(quit)
	}()

	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		log.Printf("[runner] connecting to relay as %s ...", cfg.ServerID)

		// Use a context that cancels when quit is closed.
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-quit:
				cancel()
			case <-ctx.Done():
			}
		}()

		err := connect(ctx, cfg)
		cancel()

		// Check if we were signalled to stop (quit is a closed channel — always readable).
		select {
		case <-quit:
			log.Printf("[runner] shutting down")
			return
		default:
		}

		if err != nil {
			log.Printf("[runner] connection error: %v — retrying in %s", err, backoff)
		} else {
			log.Printf("[runner] disconnected — retrying in %s", backoff)
			// Reset backoff on clean disconnect (indicates successful connection)
			backoff = 2 * time.Second
			continue
		}
		time.Sleep(backoff)
		backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
	}
}

func connect(ctx context.Context, cfg Config) error {
	wsURL := cfg.RelayURL + "/ws/server/" + cfg.ServerID
	headers := http.Header{"Authorization": {"Bearer " + cfg.Token}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	cw := &connWriter{conn: conn}

	log.Printf("[runner] connected to relay")

	// Respond to server-initiated WebSocket ping frames automatically
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	var (
		ptmx      *os.File
		ptmxMu    sync.Mutex
		busyMu    sync.Mutex
		busy      bool
		sessWr    *session.Writer
		stop      = make(chan struct{})
		pendingWs *pty.Winsize // stored if resize arrives before PTY is ready
	)

	// Send WebSocket protocol-level ping frames every 25s to keep the
	// Cloudflare tunnel alive (idle timeout ~100s). Using control frames
	// (opcode 0x9) rather than JSON so the proxy recognises them.
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				deadline := time.Now().Add(10 * time.Second)
				if err := cw.writeControl(websocket.PingMessage, nil, deadline); err != nil {
					return
				}
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Watch for context cancellation (signal)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-stop:
		}
	}()

	defer close(stop)

	for {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		var msg Msg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "client_joined":
			busyMu.Lock()
			if busy {
				busyMu.Unlock()
				cw.writeJSON(Msg{Type: "busy"})
				continue
			}
			busy = true
			busyMu.Unlock()

			go func() {
				defer func() {
					busyMu.Lock()
					busy = false
					busyMu.Unlock()
				}()
				runSession(cw, cfg, &ptmx, &ptmxMu, &sessWr, &pendingWs)
			}()

		case "input":
			ptmxMu.Lock()
			if ptmx != nil {
				ptmx.Write([]byte(msg.Data))
			}
			ptmxMu.Unlock()

		case "resize":
			ptmxMu.Lock()
			if ptmx != nil && msg.Rows > 0 && msg.Cols > 0 {
				pty.Setsize(ptmx, &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols})
			} else if msg.Rows > 0 && msg.Cols > 0 {
				pendingWs = &pty.Winsize{Rows: msg.Rows, Cols: msg.Cols}
			}
			ptmxMu.Unlock()

		case "end":
			ptmxMu.Lock()
			if ptmx != nil {
				// Send SIGHUP to the shell's process group (proper terminal hangup)
				signalPtyProcess(ptmx)
			}
			ptmxMu.Unlock()

		case "client_left":
			ptmxMu.Lock()
			if ptmx != nil {
				signalPtyProcess(ptmx)
			}
			ptmxMu.Unlock()

		case "pong":
			// protocol-level pong handled by SetPongHandler above
		}
	}
}

// signalPtyProcess sends SIGHUP to the shell process to cleanly terminate the session.
func signalPtyProcess(ptmx *os.File) {
	// Write EOF (Ctrl-D) to the PTY to signal end-of-input, then follow up
	// with SIGHUP via the PTY file descriptor. This is cleaner than injecting
	// "exit\n" which appears in the terminal output and session recording.
	ptmx.Write([]byte{0x04}) // Ctrl-D (EOF)
}

func runSession(cw *connWriter, cfg Config, ptmxPtr **os.File, mu *sync.Mutex, sessWr **session.Writer, pendingWs **pty.Winsize) {
	shell := cfg.Shell
	if shell == "" {
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
	}

	// Validate shell path exists and is executable
	if info, err := os.Stat(shell); err != nil {
		log.Printf("[runner] shell not found: %s (%v)", shell, err)
		cw.writeJSON(Msg{Type: "session_end", ExitCode: -1})
		return
	} else if info.IsDir() || info.Mode()&0111 == 0 {
		log.Printf("[runner] shell is not executable: %s", shell)
		cw.writeJSON(Msg{Type: "session_end", ExitCode: -1})
		return
	}

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("[runner] pty start error: %v", err)
		cw.writeJSON(Msg{Type: "session_end", ExitCode: -1})
		return
	}
	defer ptmx.Close()

	mu.Lock()
	*ptmxPtr = ptmx
	// Apply any resize that arrived before the PTY was ready
	if *pendingWs != nil {
		pty.Setsize(ptmx, *pendingWs)
		*pendingWs = nil
	}
	mu.Unlock()

	// Create session transcript writer — use a human-readable timestamp as the
	// filename so sessions/*.cast files are easy to identify.
	sessionID := time.Now().Format("2006-01-02_15-04-05")
	sw, err := session.New(sessionID, cfg.ServerID)
	if err != nil {
		log.Printf("[runner] session writer error: %v", err)
	}
	*sessWr = sw

	cw.writeJSON(Msg{Type: "session_start"})
	log.Printf("[runner] session started (shell=%s)", shell)

	// Stream PTY output → relay (16KB buffer for better throughput)
	buf := make([]byte, 16384)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			chunk := string(buf[:n])
			cw.writeJSON(Msg{Type: "output", Data: chunk})
			if sw != nil {
				sw.Write(chunk)
			}
		}
		if err != nil {
			// On Linux (and inside Docker), PTY master reads return EIO
			// when the slave side is closed (child process exits). This is
			// normal POSIX behaviour — treat it the same as EOF.
			if err != io.EOF && !errors.Is(err, syscall.EIO) {
				log.Printf("[runner] pty read: %v", err)
			}
			break
		}
	}

	// Wait for cmd with timeout to prevent hanging indefinitely
	exitCode := 0
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case err := <-waitDone:
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
	case <-time.After(10 * time.Second):
		log.Printf("[runner] cmd.Wait() timed out, killing process")
		cmd.Process.Kill()
		exitCode = -1
	}

	if sw != nil {
		sw.Close()
		log.Printf("[runner] session saved to %s", sw.Path())
	}

	mu.Lock()
	*ptmxPtr = nil
	mu.Unlock()

	cw.writeJSON(Msg{Type: "session_end", ExitCode: exitCode})
	log.Printf("[runner] session ended (exit=%d)", exitCode)
}
