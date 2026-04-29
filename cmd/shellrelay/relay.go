package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ShellRelay/runner/internal/config"
)

func cmdRelay(args []string) {
	fs := flag.NewFlagSet("relay", flag.ExitOnError)
	fs.Usage = func() {
		log.SetFlags(0)
		log.Printf("Usage: shellrelay relay --url <hostname or wss://...>\n\nPersist a custom relay server URL to ~/.shellrelay/config and restart the daemon.\nIf no scheme is provided, wss:// is assumed.\n\nOptions:")
		fs.PrintDefaults()
	}
	flagURL := fs.String("url", "", "Relay server hostname or WebSocket URL (e.g. api.shellrelay.com or wss://api.shellrelay.com)")
	fs.Parse(args)

	url := *flagURL
	if url == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		fs.Usage()
		os.Exit(1)
	}

	// If no scheme is provided, default to wss://
	if !strings.HasPrefix(url, "wss://") && !strings.HasPrefix(url, "ws://") {
		url = "wss://" + url
		fmt.Printf("no scheme provided, defaulting to wss:// → %s\n", url)
	}

	if err := config.Save(config.Values{"SHELLRELAY_URL": url}); err != nil {
		log.Fatalf("[relay] failed to save config: %v", err)
	}

	cfgPath, _ := config.Path()
	fmt.Printf("relay URL saved to %s\n", cfgPath)

	pid, err := readPID()
	if err == nil && isRunning(pid) {
		fmt.Println("restarting daemon...")
		cmdRestart(nil)
	} else {
		fmt.Printf("daemon is not running — start it with: shellrelay start\n")
	}
}
