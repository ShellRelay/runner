package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveToConfig(t *testing.T) {
	home := setTempHome(t)

	saveToConfig("my-server", "sr_abc123token")

	cfgPath := filepath.Join(home, ".shellrelay", "config")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "SHELLRELAY_SERVER_ID=my-server") {
		t.Errorf("config missing SHELLRELAY_SERVER_ID=my-server; got:\n%s", content)
	}
	if !strings.Contains(content, "SHELLRELAY_TOKEN=sr_abc123token") {
		t.Errorf("config missing SHELLRELAY_TOKEN=sr_abc123token; got:\n%s", content)
	}
}

func TestSaveToConfig_Updates(t *testing.T) {
	home := setTempHome(t)

	// First write
	saveToConfig("server-a", "sr_firsttoken")
	// Second write should overwrite
	saveToConfig("server-b", "sr_secondtoken")

	cfgPath := filepath.Join(home, ".shellrelay", "config")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "SHELLRELAY_SERVER_ID=server-b") {
		t.Errorf("config should have updated server ID; got:\n%s", content)
	}
	if !strings.Contains(content, "SHELLRELAY_TOKEN=sr_secondtoken") {
		t.Errorf("config should have updated token; got:\n%s", content)
	}
	if strings.Contains(content, "server-a") {
		t.Errorf("config should not contain old server ID; got:\n%s", content)
	}
}

func TestPrintUsage_NoPanic(t *testing.T) {
	// printUsage writes to os.Stderr; just verify it doesn't panic.
	printUsage()
}
