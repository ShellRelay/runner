// Package config handles loading and saving the shellrelay runner config file.
//
// Config file location: ~/.shellrelay/config
// Format: KEY=VALUE (one per line, no quoting needed, # comments supported)
//
// Supported keys:
//
//	SHELLRELAY_URL        — relay server WebSocket URL
//	SHELLRELAY_SERVER_ID  — server ID
//	SHELLRELAY_TOKEN      — server token (sr_...)
//	SHELLRELAY_SHELL      — shell override
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configFileName = "config"

// Values holds the key-value pairs from the config file.
type Values map[string]string

// Dir returns the shellrelay config directory (~/.shellrelay).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".shellrelay"), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads the config file and returns its key-value pairs.
// Returns an empty Values (not an error) if the file doesn't exist.
func Load() (Values, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Values{}, nil
		}
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	vals := Values{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return vals, nil
}

// Save writes the given key-value pairs to the config file.
// It preserves comments and ordering from an existing file, updating matched keys
// and appending new ones. If the file doesn't exist, it creates it.
func Save(updates Values) error {
	path, err := Path()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// Read existing file lines (if any)
	var lines []string
	existing, err := os.Open(path)
	if err == nil {
		scanner := bufio.NewScanner(existing)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		existing.Close()
	}

	// Track which keys we've updated
	written := map[string]bool{}

	// Update existing lines in-place
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if newVal, found := updates[key]; found {
			lines[i] = key + "=" + newVal
			written[key] = true
		}
	}

	// Append any new keys not already in the file
	for key, val := range updates {
		if !written[key] {
			lines = append(lines, key+"="+val)
		}
	}

	// Write atomically via temp file
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	for _, line := range lines {
		fmt.Fprintln(f, line)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// Get returns a config value with the priority: flag > env > config file > default.
// Pass empty string for any source to skip it.
func Get(flagVal, envKey string, fileVals Values, fileKey, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if v, ok := fileVals[fileKey]; ok && v != "" {
		return v
	}
	return defaultVal
}
