package tcconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadParsesValidLinesOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tcconfig")
	content := "# comment\nserver=\"rdp.local\"\n1bad=ignore\nhelpdesk=\"call desk\"\nempty=\"\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write tcconfig: %v", err)
	}

	store := NewStore(path)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if got := cfg["server"]; got != "rdp.local" {
		t.Fatalf("server mismatch: got %q", got)
	}
	if got := cfg["helpdesk"]; got != "call desk" {
		t.Fatalf("helpdesk mismatch: got %q", got)
	}
	if _, ok := cfg["1bad"]; ok {
		t.Fatal("invalid key should not be loaded")
	}
}

func TestSaveWritesSortedQuotedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "tcconfig")
	store := NewStore(path)

	cfg := map[string]string{
		"helpdesk": "Desk",
		"server":   `rdp\\host`,
		"bad-key":  "ignored",
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}

	got := string(b)
	wantA := "helpdesk=\"Desk\"\n"
	wantB := "server=\"rdp\\\\\\\\host\"\n"
	if got != wantA+wantB {
		t.Fatalf("saved content mismatch:\n--- got ---\n%s\n--- want ---\n%s%s", got, wantA, wantB)
	}
}

func TestValidKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		key   string
		valid bool
	}{
		{key: "server", valid: true},
		{key: "ota_channel", valid: true},
		{key: "bad-key", valid: false},
		{key: "1bad", valid: false},
		{key: "", valid: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()
			if got := ValidKey(tc.key); got != tc.valid {
				t.Fatalf("ValidKey(%q) = %v, want %v", tc.key, got, tc.valid)
			}
		})
	}
}
