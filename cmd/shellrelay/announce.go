package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/ShellRelay/runner/internal/config"
)

// generateToken creates a random server token (sr_ + 23 random bytes hex-encoded).
func generateToken() (string, error) {
	b := make([]byte, 23)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return "sr_" + hex.EncodeToString(b), nil
}

// cmdAnnounce registers this runner with the relay server using an email address.
// The runner generates its own sr_* token and sends it to POST /servers/announce.
// The server appears as "unclaimed" in the owner's dashboard until they claim it
// by entering the token shown in the container/runner logs.
func cmdAnnounce(args []string) {
	fs := flag.NewFlagSet("announce", flag.ExitOnError)
	var (
		flagRelay = fs.String("relay", "", "Relay server HTTP URL (default: derived from SHELLRELAY_URL)")
		flagEmail = fs.String("email", "", "Owner email address")
		flagName  = fs.String("name", "", "Display name (default: server ID)")
	)
	fs.Usage = func() {
		log.SetFlags(0)
		log.Printf("Usage: shellrelay announce <server-id> --email <email> [options]\n\nSelf-register this runner with the relay server.\nThe owner must claim it in the dashboard using the token printed below.\n\nOptions:")
		fs.PrintDefaults()
		log.Printf("\nEnv vars: SHELLRELAY_URL, SHELLRELAY_EMAIL, SHELLRELAY_SERVER_NAME")
	}
	fs.Parse(args)

	// Server ID from positional arg
	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(1)
	}
	serverID := fs.Arg(0)

	// Resolve email: flag > env > error
	email := *flagEmail
	if email == "" {
		email = os.Getenv("SHELLRELAY_EMAIL")
	}
	if email == "" {
		log.Fatal("[announce] email is required: --email <email> or SHELLRELAY_EMAIL env var")
	}

	// Resolve display name: flag > env > server ID
	displayName := *flagName
	if displayName == "" {
		displayName = os.Getenv("SHELLRELAY_SERVER_NAME")
	}
	if displayName == "" {
		displayName = serverID
	}

	// Resolve relay URL: flag > env > compiled-in default (NOT from config file,
	// so upgrading the binary always picks up the correct URL).
	fileVals, _ := config.Load()
	relayWsURL := config.Get(*flagRelay, "SHELLRELAY_URL", nil, "", DefaultRelayURL)

	// Convert wss:// to https:// for the HTTP API call
	relayHTTP := relayWsURL
	relayHTTP = strings.Replace(relayHTTP, "wss://", "https://", 1)
	relayHTTP = strings.Replace(relayHTTP, "ws://", "http://", 1)

	// Check if we already have a saved token — skip announce if so
	existingToken := config.Get("", "SHELLRELAY_TOKEN", fileVals, "SHELLRELAY_TOKEN", "")
	existingID := config.Get("", "SHELLRELAY_SERVER_ID", fileVals, "SHELLRELAY_SERVER_ID", "")
	if existingToken != "" && existingID == serverID {
		fmt.Println("  Already announced — using saved token")
		fmt.Println()
		fmt.Printf("  Server ID : %s\n", serverID)
		fmt.Printf("  Owner     : %s\n", email)
		fmt.Printf("  Token     : %s\n", existingToken)
		fmt.Println()
		fmt.Println("  Log in to https://www.shellrelay.com and")
		fmt.Println("  claim this server with the token above.")
		fmt.Println()
		fmt.Println("  To re-announce, delete ~/.shellrelay/config")
		return
	}

	// Generate a new token
	token, err := generateToken()
	if err != nil {
		log.Fatalf("[announce] %v", err)
	}

	// Call POST /servers/announce
	body, _ := json.Marshal(map[string]string{
		"id":    serverID,
		"name":  displayName,
		"email": email,
		"token": token,
	})

	resp, err := http.Post(relayHTTP+"/servers/announce", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Fatalf("[announce] failed to contact relay: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(respBody, &errResp)
		if errResp.Error != "" {
			log.Fatalf("[announce] failed: %s", errResp.Error)
		}
		log.Fatalf("[announce] failed: HTTP %d — %s", resp.StatusCode, string(respBody))
	}

	// Parse the response to get the actual server ID (may differ if auto-resolved)
	var announceResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(respBody, &announceResp)
	actualID := announceResp.ID
	if actualID == "" {
		actualID = serverID // fallback
	}

	// Save config with the actual (possibly auto-resolved) server ID
	// Save config — intentionally omit SHELLRELAY_URL so the compiled-in
	// default is always used (prevents stale URLs after upgrade).
	if err := config.Save(config.Values{
		"SHELLRELAY_SERVER_ID": actualID,
		"SHELLRELAY_TOKEN":     token,
	}); err != nil {
		log.Printf("[announce] warning: could not save config: %v", err)
	}

	// Print the token prominently
	fmt.Println()
	fmt.Println("  ✓ ShellRelay server announced!")
	fmt.Println()
	fmt.Printf("  Server ID : %s\n", actualID)
	if actualID != serverID {
		fmt.Printf("  (requested: %s)\n", serverID)
	}
	fmt.Printf("  Owner     : %s\n", email)
	fmt.Printf("  Token     : %s\n", token)
	fmt.Println()
	fmt.Println("  Log in to https://www.shellrelay.com and")
	fmt.Println("  claim this server with the token above.")
	fmt.Println()
}
