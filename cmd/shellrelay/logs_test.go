package main

import (
	"os"
	"strings"
	"testing"
)

func TestPrintLastN(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		n     int
		want  []string
	}{
		{
			name:  "fewer lines than n",
			lines: []string{"a", "b", "c"},
			n:     10,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "exactly n lines",
			lines: []string{"a", "b", "c"},
			n:     3,
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "more lines than n returns tail",
			lines: []string{"a", "b", "c", "d", "e"},
			n:     3,
			want:  []string{"c", "d", "e"},
		},
		{
			name:  "n=1 returns last line only",
			lines: []string{"first", "second", "last"},
			n:     1,
			want:  []string{"last"},
		},
		{
			name:  "empty file produces no output",
			lines: []string{},
			n:     5,
			want:  []string{},
		},
		{
			name:  "ring buffer wraps correctly at 2n",
			lines: []string{"1", "2", "3", "4", "5", "6"},
			n:     3,
			want:  []string{"4", "5", "6"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Write test content to a temp file.
			f, err := os.CreateTemp(t.TempDir(), "logs_test_*.log")
			if err != nil {
				t.Fatalf("create temp: %v", err)
			}
			defer f.Close()

			if len(tc.lines) > 0 {
				if _, err := f.WriteString(strings.Join(tc.lines, "\n") + "\n"); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			if _, err := f.Seek(0, 0); err != nil {
				t.Fatalf("seek: %v", err)
			}

			// Redirect stdout to a pipe so we can capture printLastN output.
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatalf("pipe: %v", err)
			}
			origStdout := os.Stdout
			os.Stdout = w

			printLastN(f, tc.n)

			w.Close()
			os.Stdout = origStdout

			buf := make([]byte, 65536)
			n, _ := r.Read(buf)
			r.Close()

			got := strings.TrimRight(string(buf[:n]), "\n")

			if len(tc.want) == 0 {
				if got != "" {
					t.Errorf("expected empty output, got %q", got)
				}
				return
			}

			wantStr := strings.Join(tc.want, "\n")
			if got != wantStr {
				t.Errorf("printLastN:\n  got  %q\n  want %q", got, wantStr)
			}
		})
	}
}
