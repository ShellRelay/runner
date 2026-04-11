package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ShellRelay/runner/internal/config"
)

func cmdRotate(args []string) {
	fs := flag.NewFlagSet("rotate", flag.ExitOnError)
	var (
		flagToken = fs.String("token", "", "New server token (sr_...)")
	)
	fs.Usage = func() {
		log.SetFlags(0)
		log.Printf("Usage: shellrelay rotate --token <new-token>\n\nUpdate the server token in ~/.shellrelay/config and restart the daemon.\n\nOptions:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	newToken := *flagToken
	if newToken == "" {
		fmt.Fprintln(os.Stderr, "Error: --token is required")
		fs.Usage()
		os.Exit(1)
	}

	if !strings.HasPrefix(newToken, "sr_") {
		fmt.Fprintln(os.Stderr, "Error: token must start with 'sr_'")
		os.Exit(1)
	}

	// Save new token to config file
	if err := config.Save(config.Values{"SHELLRELAY_TOKEN": newToken}); err != nil {
		log.Fatalf("[rotate] failed to save config: %v", err)
	}

	cfgPath, _ := config.Path()
	log.Printf("[rotate] token updated in %s", cfgPath)

	// Restart the daemon if it is running
	pid, err := readPID()
	if err == nil && isRunning(pid) {
		cmdRestart(nil)
	} else {
		log.Printf("[rotate] daemon is not running — start it with: shellrelay start")
	}
}
