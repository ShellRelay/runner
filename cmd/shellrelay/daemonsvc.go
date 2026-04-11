package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// cmdDaemon dispatches daemon sub-commands: install, uninstall.
func cmdDaemon(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: shellrelay daemon <install|uninstall>\n\n")
		fmt.Fprintf(os.Stderr, "  install    Register shellrelay as a login service (launchd on macOS, systemd on Linux)\n")
		fmt.Fprintf(os.Stderr, "  uninstall  Remove the login service\n")
		os.Exit(1)
	}
	switch args[0] {
	case "install":
		daemonInstall()
	case "uninstall":
		daemonUninstall()
	default:
		fmt.Fprintf(os.Stderr, "Unknown daemon subcommand: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage: shellrelay daemon <install|uninstall>\n")
		os.Exit(1)
	}
}

// ── macOS launchd ─────────────────────────────────────────────────────────────

const plistLabel = "com.shellrelay.runner"

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.shellrelay.runner</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.Executable}}</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
    <key>ThrottleInterval</key>
    <integer>10</integer>
</dict>
</plist>
`))

// ── Linux systemd ─────────────────────────────────────────────────────────────

var serviceTemplate = template.Must(template.New("service").Parse(`[Unit]
Description=ShellRelay Runner
Documentation=https://github.com/ShellRelay/runner
After=network-online.target
Wants=network-online.target

[Service]
ExecStart={{.Executable}} run
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
`))

// ── install ───────────────────────────────────────────────────────────────────

func daemonInstall() {
	self, err := os.Executable()
	if err != nil {
		fatalf("daemon install: cannot determine executable path: %v", err)
	}
	// Resolve symlinks so the plist/service points to the real binary.
	if resolved, err := filepath.EvalSymlinks(self); err == nil {
		self = resolved
	}

	logPath, err := logFile()
	if err != nil {
		fatalf("daemon install: %v", err)
	}

	data := struct {
		Executable string
		LogPath    string
	}{self, logPath}

	switch runtime.GOOS {
	case "darwin":
		installLaunchd(data)
	case "linux":
		installSystemd(data)
	default:
		fatalf("daemon install: unsupported OS %q (supported: darwin, linux)", runtime.GOOS)
	}
}

func installLaunchd(data struct{ Executable, LogPath string }) {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("daemon install: cannot determine home directory: %v", err)
	}
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		fatalf("daemon install: create LaunchAgents dir: %v", err)
	}
	plistPath := filepath.Join(agentsDir, plistLabel+".plist")

	// Unload existing service first (ignore errors — may not be loaded)
	exec.Command("launchctl", "unload", plistPath).Run() //nolint:errcheck

	f, err := os.OpenFile(plistPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fatalf("daemon install: write plist: %v", err)
	}
	if err := plistTemplate.Execute(f, data); err != nil {
		f.Close()
		fatalf("daemon install: render plist: %v", err)
	}
	f.Close()

	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		fatalf("daemon install: launchctl load: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	fmt.Printf("shellrelay daemon installed (launchd)\n")
	fmt.Printf("plist:      %s\n", plistPath)
	fmt.Printf("binary:     %s\n", data.Executable)
	fmt.Printf("log:        %s\n", data.LogPath)
	fmt.Printf("\nThe runner will start automatically at login and restart on exit.\n")
	fmt.Printf("To remove: shellrelay daemon uninstall\n")
}

func installSystemd(data struct{ Executable, LogPath string }) {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("daemon install: cannot determine home directory: %v", err)
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		fatalf("daemon install: create systemd user dir: %v", err)
	}
	unitPath := filepath.Join(unitDir, "shellrelay.service")

	f, err := os.OpenFile(unitPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fatalf("daemon install: write service file: %v", err)
	}
	if err := serviceTemplate.Execute(f, data); err != nil {
		f.Close()
		fatalf("daemon install: render service: %v", err)
	}
	f.Close()

	run := func(name string, args ...string) {
		if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
			fatalf("daemon install: %s %s: %v\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}

	run("systemctl", "--user", "daemon-reload")
	run("systemctl", "--user", "enable", "shellrelay")
	run("systemctl", "--user", "restart", "shellrelay")

	fmt.Printf("shellrelay daemon installed (systemd)\n")
	fmt.Printf("unit file:  %s\n", unitPath)
	fmt.Printf("binary:     %s\n", data.Executable)
	fmt.Printf("\nThe runner will start automatically at login and restart on exit.\n")
	fmt.Printf("To remove: shellrelay daemon uninstall\n")
	fmt.Printf("\nTip: to start on boot (not just login), run:\n")
	fmt.Printf("  loginctl enable-linger %s\n", os.Getenv("USER"))
}

// ── uninstall ─────────────────────────────────────────────────────────────────

func daemonUninstall() {
	switch runtime.GOOS {
	case "darwin":
		uninstallLaunchd()
	case "linux":
		uninstallSystemd()
	default:
		fatalf("daemon uninstall: unsupported OS %q", runtime.GOOS)
	}
}

func uninstallLaunchd() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("daemon uninstall: cannot determine home directory: %v", err)
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", plistLabel+".plist")

	out, err := exec.Command("launchctl", "unload", plistPath).CombinedOutput()
	if err != nil && !strings.Contains(string(out), "No such file") {
		fmt.Fprintf(os.Stderr, "warning: launchctl unload: %v — %s\n", err, strings.TrimSpace(string(out)))
	}

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		fatalf("daemon uninstall: remove plist: %v", err)
	}

	fmt.Printf("shellrelay daemon uninstalled (launchd)\n")
	fmt.Printf("removed: %s\n", plistPath)
}

func uninstallSystemd() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("daemon uninstall: cannot determine home directory: %v", err)
	}
	unitPath := filepath.Join(home, ".config", "systemd", "user", "shellrelay.service")

	exec.Command("systemctl", "--user", "stop", "shellrelay").Run()    //nolint:errcheck
	exec.Command("systemctl", "--user", "disable", "shellrelay").Run() //nolint:errcheck

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		fatalf("daemon uninstall: remove unit file: %v", err)
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run() //nolint:errcheck

	fmt.Printf("shellrelay daemon uninstalled (systemd)\n")
	fmt.Printf("removed: %s\n", unitPath)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}
