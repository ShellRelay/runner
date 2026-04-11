package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const githubRepo = "ShellRelay/runner"

func cmdUpgrade(args []string) {
	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for latest release...")

	latest, downloadURL, checksumURL, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: %v\n", err)
		os.Exit(1)
	}

	// Use semantic version comparison instead of string comparison
	currentSemver := "v" + Version
	if compareSemver(latest, currentSemver) <= 0 {
		fmt.Printf("Already up to date (%s)\n", Version)
		return
	}

	fmt.Printf("New version available: %s → %s\n", Version, latest)
	fmt.Printf("Downloading %s ...\n", downloadURL)

	// Download new binary to a temp file beside the current executable
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: cannot determine executable path: %v\n", err)
		os.Exit(1)
	}

	tmp := self + ".new"
	if err := downloadFile(tmp, downloadURL); err != nil {
		os.Remove(tmp)
		fmt.Fprintf(os.Stderr, "upgrade: download failed: %v\n", err)
		os.Exit(1)
	}

	// Verify checksum if available
	if checksumURL != "" {
		if err := verifyChecksum(tmp, checksumURL); err != nil {
			os.Remove(tmp)
			fmt.Fprintf(os.Stderr, "upgrade: checksum verification failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Checksum verified.")
	}

	if err := os.Chmod(tmp, 0755); err != nil {
		os.Remove(tmp)
		fmt.Fprintf(os.Stderr, "upgrade: chmod: %v\n", err)
		os.Exit(1)
	}

	// Backup old binary before replacing
	backup := self + ".bak"
	if err := copyFile(self, backup); err != nil {
		fmt.Fprintf(os.Stderr, "upgrade: warning: could not backup old binary: %v\n", err)
		// Continue anyway — not fatal
	}

	// Atomically replace the running binary
	if err := os.Rename(tmp, self); err != nil {
		os.Remove(tmp)
		fmt.Fprintf(os.Stderr, "upgrade: replace binary: %v\n", err)
		fmt.Fprintf(os.Stderr, "       Try: sudo shellrelay upgrade\n")
		// Try to restore backup
		if _, statErr := os.Stat(backup); statErr == nil {
			os.Rename(backup, self)
		}
		os.Exit(1)
	}
	os.Remove(backup) // clean up backup on success

	fmt.Printf("Upgraded to %s\n", latest)

	// Restart the daemon if it was running
	pid, _ := readPID()
	if isRunning(pid) {
		fmt.Println("Restarting daemon...")
		cmdRestart(nil)
	}
}

// fetchLatestRelease queries the GitHub releases API and returns the tag,
// the download URL for the current platform's binary, and the checksums URL.
func fetchLatestRelease() (tag, downloadURL, checksumURL string, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	resp, err := http.Get(url)
	if err != nil {
		return "", "", "", fmt.Errorf("fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", "", fmt.Errorf("parse release: %w", err)
	}

	tag = release.TagName

	// Find asset matching current OS/arch: shellrelay-<os>-<arch>
	wantName := fmt.Sprintf("shellrelay-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, a := range release.Assets {
		if a.Name == wantName {
			downloadURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		return "", "", "", fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, tag)
	}

	return tag, downloadURL, checksumURL, nil
}

// verifyChecksum downloads the checksums file and verifies the downloaded binary matches.
func verifyChecksum(filePath, checksumURL string) error {
	// Calculate SHA256 of the downloaded file
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))

	// Download and parse checksums file
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	wantName := fmt.Sprintf("shellrelay-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == wantName {
			if parts[0] != actual {
				return fmt.Errorf("checksum mismatch: expected %s, got %s", parts[0], actual)
			}
			return nil
		}
	}

	return fmt.Errorf("no checksum found for %s in checksums file", wantName)
}

// downloadFile downloads url and writes the content to dest.
func downloadFile(dest, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// compareSemver compares two version strings like "v1.2.8" and "v1.3.0".
// Returns -1 if a < b, 0 if a == b, 1 if a > b. Falls back to string comparison.
func compareSemver(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}
