package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ShellRelay/runner/internal/config"
)

// pidFile returns the path to the PID file (~/.shellrelay/shellrelay.pid).
func pidFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shellrelay.pid"), nil
}

// logFile returns the path to the daemon log file (~/.shellrelay/shellrelay.log).
func logFile() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shellrelay.log"), nil
}

// readPID reads the PID from the PID file. Returns 0 if not found.
func readPID() (int, error) {
	path, err := pidFile()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, nil
}

// writePID writes the given PID to the PID file.
func writePID(pid int) error {
	path, err := pidFile()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0600)
}

// removePID deletes the PID file.
func removePID() {
	path, _ := pidFile()
	os.Remove(path)
}

// isRunning returns true if the PID is alive.
func isRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0: check existence without sending a signal
	err = proc.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

// cmdStart starts shellrelay as a background daemon.
func cmdStart(args []string) {
	// If we are the daemon child (re-invoked with --daemon-child), run the relay directly.
	if len(args) > 0 && args[0] == "--daemon-child" {
		runDaemonChild(args[1:])
		return
	}

	// Parse flags — positional args <server-id> [<token>] may appear before or after flags.
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	noDaemon := fs.Bool("no-daemon", false, "Skip registering as a login service (launchd/systemd)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: shellrelay start [<server-id> [<token>]] [--no-daemon]\n\n")
		fmt.Fprintf(os.Stderr, "Start the runner as a background daemon and register it as a login service.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	positional := fs.Args()

	// Persist server-id / token to config if provided as positional args.
	// Intentionally omit SHELLRELAY_URL — the compiled-in default is always
	// used so upgrading the binary automatically picks up URL changes.
	if len(positional) >= 2 {
		serverID := positional[0]
		token := positional[1]
		updates := config.Values{
			"SHELLRELAY_SERVER_ID": serverID,
			"SHELLRELAY_TOKEN":     token,
		}
		if err := config.Save(updates); err != nil {
			log.Printf("start: warning: could not save config: %v", err)
		} else {
			fmt.Printf("config saved (server=%s)\n", serverID)
		}
	} else if len(positional) == 1 {
		serverID := positional[0]
		if err := config.Save(config.Values{
			"SHELLRELAY_SERVER_ID": serverID,
		}); err != nil {
			log.Printf("start: warning: could not save config: %v", err)
		}
	}

	pid, err := readPID()
	if err == nil && isRunning(pid) {
		fmt.Fprintf(os.Stderr, "shellrelay is already running (pid %d)\n", pid)
		os.Exit(1)
	}

	logPath, err := logFile()
	if err != nil {
		log.Fatalf("start: %v", err)
	}

	// Build the child command: re-invoke ourselves with "start --daemon-child"
	// No extra args needed — child reads server ID and token from config file.
	self, err := os.Executable()
	if err != nil {
		log.Fatalf("start: cannot determine executable path: %v", err)
	}

	cmd := exec.Command(self, "start", "--daemon-child")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from terminal

	lf, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.Fatalf("start: open log file: %v", err)
	}
	cmd.Stdout = lf
	cmd.Stderr = lf
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		lf.Close()
		log.Fatalf("start: fork failed: %v", err)
	}
	lf.Close()

	if err := writePID(cmd.Process.Pid); err != nil {
		log.Fatalf("start: write PID: %v", err)
	}

	fmt.Printf("shellrelay started (pid %d)\n", cmd.Process.Pid)
	fmt.Printf("logs: %s\n", logPath)

	// Register as a login service so the runner auto-restarts on reboot,
	// unless the user explicitly opted out with --no-daemon.
	if !*noDaemon {
		fmt.Println()
		daemonInstall()
	}
}

// runDaemonChild is the actual daemon logic running in the background process.
func runDaemonChild(args []string) {
	// Remove PID file on exit
	defer removePID()

	// Re-use cmdRun to connect and serve
	cmdRun(args)
}

// cmdStop stops the background daemon.
func cmdStop(_ []string) {
	pid, err := readPID()
	if err != nil {
		log.Fatalf("stop: %v", err)
	}
	if pid == 0 || !isRunning(pid) {
		fmt.Println("shellrelay is not running")
		removePID()
		return
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("stop: %v", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		log.Fatalf("stop: send SIGTERM: %v", err)
	}

	// Wait for process to actually exit (up to 10 seconds)
	for i := 0; i < 40; i++ {
		if !isRunning(pid) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	// Force kill if still running
	if isRunning(pid) {
		proc.Signal(syscall.SIGKILL)
		time.Sleep(500 * time.Millisecond)
	}

	removePID()
	fmt.Printf("shellrelay stopped (pid %d)\n", pid)
}

// cmdRestart stops then starts the daemon.
func cmdRestart(args []string) {
	pid, err := readPID()
	if err == nil && isRunning(pid) {
		proc, _ := os.FindProcess(pid)
		if proc != nil {
			proc.Signal(syscall.SIGTERM)
		}
		// Wait for process to exit before restarting
		for i := 0; i < 40; i++ {
			if !isRunning(pid) {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		if isRunning(pid) {
			proc.Signal(syscall.SIGKILL)
			time.Sleep(500 * time.Millisecond)
		}
		removePID()
		fmt.Printf("shellrelay stopped (pid %d)\n", pid)
	}
	cmdStart(args)
}

// cmdStatus prints whether the daemon is running.
func cmdStatus(_ []string) {
	pid, err := readPID()
	if err != nil {
		log.Fatalf("status: %v", err)
	}
	if pid == 0 || !isRunning(pid) {
		fmt.Println("shellrelay is stopped")
		removePID()
		os.Exit(1)
	}
	logPath, _ := logFile()
	fmt.Printf("shellrelay is running (pid %d)\n", pid)
	fmt.Printf("logs: %s\n", logPath)
}
