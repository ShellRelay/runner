package main

import (
	"fmt"
	"os"
)

// Version is set at build time via -ldflags
var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		cmdStart(os.Args[2:])
	case "stop":
		cmdStop(os.Args[2:])
	case "restart":
		cmdRestart(os.Args[2:])
	case "status":
		cmdStatus(os.Args[2:])
	case "logs":
		cmdLogs(os.Args[2:])
	case "sessions":
		cmdSessions(os.Args[2:])
	case "upgrade":
		cmdUpgrade(os.Args[2:])
	case "daemon":
		cmdDaemon(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "announce":
		cmdAnnounce(os.Args[2:])
	case "rotate":
		cmdRotate(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("shellrelay v%s\n", Version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: shellrelay <command> [options]

Quick start:
  shellrelay start <server-id> <token>

Daemon:
  start              Start the runner as a background daemon
  stop               Stop the background daemon
  restart            Restart the background daemon
  status             Show whether the daemon is running and its PID
  logs               Show daemon log output  (-f to follow, -n <lines>)
  sessions           List locally recorded terminal sessions
  upgrade            Download and install the latest release
  daemon install     Register as a login service (auto-start on reboot)
  daemon uninstall   Remove the login service

Docker / self-registration:
  announce   Self-register with the relay (runner generates its own token)

Advanced:
  run       Run in the foreground (no daemon)
  rotate    Rotate the server token and restart the daemon

Other:
  version   Print version and exit
  help      Show this help message

Run 'shellrelay <command> --help' for more information on a command.
`)
}
