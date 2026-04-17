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
		log.Printf("Usage: shellrelay relay --url <wss://...>\n\nPersist a custom relay server URL to ~/.shellrelay/config and restart the daemon.\n\nOptions:")
		fs.PrintDefaults()
	}
	flagURL := fs.String("url", "", "Relay server WebSocket URL (wss://... or ws://...)")
	fs.Parse(args)

	url := *flagURL
	if url == "" {
		fmt.Fprintln(os.Stderr, "Error: --url is required")
		fs.Usage()
		os.Exit(1)
	}

	if !strings.HasPrefix(url, "wss://") && !strings.HasPrefix(url, "ws://") {
		fmt.Fprintln(os.Stderr, "Error: URL must start with wss:// or ws://")
		os.Exit(1)
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
