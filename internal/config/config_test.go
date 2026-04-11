package config

import (
	"os"
	"path/filepath"
	"testing"
)

// helpers -----------------------------------------------------------------

// withHome overrides HOME so Dir()/Path()/Load()/Save() use a temp directory.
func withHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
}

// writeFile is a small helper to create a file with content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// -------------------------------------------------------------------------
// Dir / Path
// -------------------------------------------------------------------------

func TestDir(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	want := filepath.Join(tmp, ".shellrelay")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestPath(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path() error: %v", err)
	}
	want := filepath.Join(tmp, ".shellrelay", "config")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

// -------------------------------------------------------------------------
// Load
// -------------------------------------------------------------------------

func TestLoad_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("Load() returned %d entries for missing file, want 0", len(vals))
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"), "")

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(vals) != 0 {
		t.Errorf("Load() = %v, want empty map", vals)
	}
}

func TestLoad_KeyValuePairs(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"),
		"SHELLRELAY_URL=wss://example.com\nSHELLRELAY_TOKEN=sr_abc123\n")

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := vals["SHELLRELAY_URL"]; got != "wss://example.com" {
		t.Errorf("SHELLRELAY_URL = %q, want %q", got, "wss://example.com")
	}
	if got := vals["SHELLRELAY_TOKEN"]; got != "sr_abc123" {
		t.Errorf("SHELLRELAY_TOKEN = %q, want %q", got, "sr_abc123")
	}
}

func TestLoad_CommentsAndBlanks(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	content := `# this is a comment
SHELLRELAY_URL=wss://relay.example.com

# another comment
SHELLRELAY_TOKEN=sr_tok

`
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"), content)

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("Load() returned %d entries, want 2", len(vals))
	}
	if got := vals["SHELLRELAY_URL"]; got != "wss://relay.example.com" {
		t.Errorf("SHELLRELAY_URL = %q", got)
	}
	if got := vals["SHELLRELAY_TOKEN"]; got != "sr_tok" {
		t.Errorf("SHELLRELAY_TOKEN = %q", got)
	}
}

func TestLoad_MalformedLines(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	content := "GOODKEY=goodval\nno_equals_here\nANOTHER=val2\n"
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"), content)

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("Load() returned %d entries, want 2 (malformed line skipped)", len(vals))
	}
}

func TestLoad_WhitespaceAroundKeyValue(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"),
		"  KEY  =  value  \n")

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := vals["KEY"]; got != "value" {
		t.Errorf("KEY = %q, want %q", got, "value")
	}
}

// -------------------------------------------------------------------------
// Save
// -------------------------------------------------------------------------

func TestSave_CreatesNewFile(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	err := Save(Values{"SHELLRELAY_URL": "wss://new.example.com"})
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := vals["SHELLRELAY_URL"]; got != "wss://new.example.com" {
		t.Errorf("after Save, SHELLRELAY_URL = %q", got)
	}
}

func TestSave_UpdatesExistingKey(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"),
		"SHELLRELAY_URL=old\nSHELLRELAY_TOKEN=keep\n")

	err := Save(Values{"SHELLRELAY_URL": "new"})
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := vals["SHELLRELAY_URL"]; got != "new" {
		t.Errorf("SHELLRELAY_URL = %q, want %q", got, "new")
	}
	if got := vals["SHELLRELAY_TOKEN"]; got != "keep" {
		t.Errorf("SHELLRELAY_TOKEN = %q, want %q (should be preserved)", got, "keep")
	}
}

func TestSave_PreservesComments(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	original := "# My config\nSHELLRELAY_URL=old\n"
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"), original)

	if err := Save(Values{"SHELLRELAY_URL": "new"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(tmp, ".shellrelay", "config"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(raw)
	if !contains(content, "# My config") {
		t.Error("Save() did not preserve comment line")
	}
	if !contains(content, "SHELLRELAY_URL=new") {
		t.Error("Save() did not write updated key")
	}
}

func TestSave_AppendsNewKey(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)
	writeFile(t, filepath.Join(tmp, ".shellrelay", "config"),
		"EXISTING=val\n")

	if err := Save(Values{"NEW_KEY": "new_val"}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	vals, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if got := vals["EXISTING"]; got != "val" {
		t.Errorf("EXISTING = %q", got)
	}
	if got := vals["NEW_KEY"]; got != "new_val" {
		t.Errorf("NEW_KEY = %q, want %q", got, "new_val")
	}
}

// -------------------------------------------------------------------------
// Save then Load round-trip
// -------------------------------------------------------------------------

func TestSaveLoad_RoundTrip(t *testing.T) {
	tmp := t.TempDir()
	withHome(t, tmp)

	original := Values{
		"SHELLRELAY_URL":       "wss://relay.io",
		"SHELLRELAY_SERVER_ID": "srv-42",
		"SHELLRELAY_TOKEN":     "sr_roundtrip",
		"SHELLRELAY_SHELL":     "/bin/zsh",
	}
	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	for k, want := range original {
		if got := loaded[k]; got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

// -------------------------------------------------------------------------
// Get — priority resolution
// -------------------------------------------------------------------------

func TestGet(t *testing.T) {
	tests := []struct {
		name       string
		flagVal    string
		envKey     string
		envVal     string
		fileVals   Values
		fileKey    string
		defaultVal string
		want       string
	}{
		{
			name:       "flag wins over all",
			flagVal:    "from-flag",
			envKey:     "TEST_GET_ENV",
			envVal:     "from-env",
			fileVals:   Values{"K": "from-file"},
			fileKey:    "K",
			defaultVal: "from-default",
			want:       "from-flag",
		},
		{
			name:       "env wins when flag empty",
			flagVal:    "",
			envKey:     "TEST_GET_ENV",
			envVal:     "from-env",
			fileVals:   Values{"K": "from-file"},
			fileKey:    "K",
			defaultVal: "from-default",
			want:       "from-env",
		},
		{
			name:       "file wins when flag and env empty",
			flagVal:    "",
			envKey:     "TEST_GET_NOENV",
			envVal:     "",
			fileVals:   Values{"K": "from-file"},
			fileKey:    "K",
			defaultVal: "from-default",
			want:       "from-file",
		},
		{
			name:       "default when all empty",
			flagVal:    "",
			envKey:     "TEST_GET_NOENV2",
			envVal:     "",
			fileVals:   Values{},
			fileKey:    "K",
			defaultVal: "from-default",
			want:       "from-default",
		},
		{
			name:       "file key present but empty falls through to default",
			flagVal:    "",
			envKey:     "TEST_GET_NOENV3",
			envVal:     "",
			fileVals:   Values{"K": ""},
			fileKey:    "K",
			defaultVal: "fallback",
			want:       "fallback",
		},
		{
			name:       "file key missing falls through to default",
			flagVal:    "",
			envKey:     "TEST_GET_NOENV4",
			envVal:     "",
			fileVals:   Values{"OTHER": "val"},
			fileKey:    "K",
			defaultVal: "fallback",
			want:       "fallback",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVal != "" {
				t.Setenv(tc.envKey, tc.envVal)
			} else {
				// Ensure the env var is unset for this test.
				t.Setenv(tc.envKey, "")
				os.Unsetenv(tc.envKey)
			}
			got := Get(tc.flagVal, tc.envKey, tc.fileVals, tc.fileKey, tc.defaultVal)
			if got != tc.want {
				t.Errorf("Get() = %q, want %q", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------------------------
// helper
// -------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}
func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
