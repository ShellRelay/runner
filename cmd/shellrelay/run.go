package main

import (
	"flag"
	"log"

	"github.com/ShellRelay/runner/internal/config"
	"github.com/ShellRelay/runner/internal/relay"
)

const DefaultRelayURL = "wss://prod-api.shellrelay.com"

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	var (
		flagRelay = fs.String("relay", "", "Relay server WebSocket URL")
		flagID    = fs.String("id", "", "Server ID (e.g. my-macbook)")
		flagToken = fs.String("token", "", "Server token (sr_...)")
		flagShell = fs.String("shell", "", "Shell to use (default: $SHELL)")
	)
	fs.Usage = func() {
		log.SetFlags(0)
		log.Printf("Usage: shellrelay run [<server-id> <token>] [options]\n\nConnect to the relay server and serve a terminal.\n\nPositional args:\n  shellrelay run <server-id> <token>    Quick start\n\nOptions:")
		fs.PrintDefaults()
		log.Printf("\nPriority: positional args > flags > env vars > ~/.shellrelay/config > defaults")
		log.Printf("\nEnv vars: SHELLRELAY_URL, SHELLRELAY_SERVER_ID, SHELLRELAY_TOKEN, SHELLRELAY_SHELL")
	}
	fs.Parse(args)

	// Support positional args: shellrelay run <server-id> <token>
	positionalID := ""
	positionalToken := ""
	if fs.NArg() >= 2 {
		positionalID = fs.Arg(0)
		positionalToken = fs.Arg(1)
	} else if fs.NArg() == 1 {
		positionalID = fs.Arg(0)
	}

	// Load config file
	fileVals, err := config.Load()
	if err != nil {
		log.Printf("[runner] warning: could not load config: %v", err)
		fileVals = config.Values{}
	}

	// Resolve relay URL: flag > env > compiled-in default (NOT from config file,
	// so upgrading the binary always picks up the correct URL).
	relayURL := config.Get(*flagRelay, "SHELLRELAY_URL", nil, "", DefaultRelayURL)
	serverID := config.Get(positionalID, "", nil, "", config.Get(*flagID, "SHELLRELAY_SERVER_ID", fileVals, "SHELLRELAY_SERVER_ID", ""))
	token := config.Get(positionalToken, "", nil, "", config.Get(*flagToken, "SHELLRELAY_TOKEN", fileVals, "SHELLRELAY_TOKEN", ""))
	shell := config.Get(*flagShell, "SHELLRELAY_SHELL", fileVals, "SHELLRELAY_SHELL", "")

	if serverID == "" {
		log.Fatal("[runner] server ID is required: shellrelay run <server-id> <token>")
	}
	if token == "" {
		log.Fatal("[runner] token is required: shellrelay run <server-id> <token>")
	}

	// Persist resolved values to ~/.shellrelay/config so future invocations
	// (including `shellrelay start`) work without repeating the arguments.
	saveToConfig(serverID, token)

	log.Printf("[runner] shellrelay v%s starting (server=%s, relay=%s)", Version, serverID, relayURL)

	relay.Run(relay.Config{
		RelayURL: relayURL,
		ServerID: serverID,
		Token:    token,
		Shell:    shell,
	})
}

// saveToConfig persists server ID and token to ~/.shellrelay/config.
// The relay URL is intentionally NOT saved — it comes from the compiled-in
// default so that upgrading the binary automatically picks up URL changes.
// Silent on error — a warning is printed but execution continues.
func saveToConfig(serverID, token string) {
	updates := config.Values{
		"SHELLRELAY_SERVER_ID": serverID,
		"SHELLRELAY_TOKEN":     token,
	}
	if err := config.Save(updates); err != nil {
		log.Printf("[runner] warning: could not save config: %v", err)
	} else {
		log.Printf("[runner] config saved to ~/.shellrelay/config")
	}
}
