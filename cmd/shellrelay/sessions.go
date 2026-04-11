package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// castHeader is the first line of an asciinema v2 .cast file.
type castHeader struct {
	Version   int    `json:"version"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Timestamp int64  `json:"timestamp"`
	Title     string `json:"title"`
}

func cmdSessions(args []string) {
	fs := flag.NewFlagSet("sessions", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: shellrelay sessions\n\nList recorded sessions stored in ~/.shellrelay/sessions/\n")
	}
	fs.Parse(args)

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sessions: %v\n", err)
		os.Exit(1)
	}
	dir := filepath.Join(home, ".shellrelay", "sessions")

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sessions: read dir: %v\n", err)
		os.Exit(1)
	}

	// Collect .cast files
	type sessionInfo struct {
		filename string
		ts       int64
		duration float64
		size     int64
		title    string
	}
	var sessions []sessionInfo

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cast") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, _ := e.Info()
		si := sessionInfo{filename: e.Name(), size: info.Size()}

		// Parse header for timestamp + title
		f, err := os.Open(path)
		if err == nil {
			var hdr castHeader
			dec := json.NewDecoder(f)
			if err := dec.Decode(&hdr); err == nil {
				si.ts = hdr.Timestamp
				si.title = hdr.Title
			}
			// Scan to last event line for duration
			si.duration = castDuration(path)
			f.Close()
		}
		sessions = append(sessions, si)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions recorded yet.")
		return
	}

	// Sort newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ts > sessions[j].ts
	})

	fmt.Printf("%-26s  %-8s  %-8s  %s\n", "STARTED", "DURATION", "SIZE", "FILE")
	fmt.Println(strings.Repeat("─", 70))
	for _, s := range sessions {
		started := "unknown"
		if s.ts > 0 {
			started = time.Unix(s.ts, 0).Format("2006-01-02 15:04:05")
		}
		dur := "?"
		if s.duration > 0 {
			d := time.Duration(s.duration * float64(time.Second))
			dur = fmtDuration(d)
		}
		sz := fmtSize(s.size)
		fmt.Printf("%-26s  %-8s  %-8s  %s\n", started, dur, sz, s.filename)
	}
	fmt.Printf("\n%d session(s) in %s\n", len(sessions), dir)
}

// castDuration reads the last event line of a .cast file and returns its timestamp.
func castDuration(path string) float64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Skip header line, then scan for the last event
	var last float64
	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] != '[' {
			continue
		}
		// Event format: [<elapsed>,"o","..."]
		var ev [3]json.RawMessage
		if err := json.Unmarshal([]byte(line), &ev); err == nil {
			var t float64
			if json.Unmarshal(ev[0], &t) == nil {
				last = t
			}
		}
	}
	return last
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func fmtSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}
