package session

import "testing"

func TestConnectRequiresServer(t *testing.T) {
	t.Parallel()

	m := NewManager("xfreerdp3", t.TempDir()+"/session.log")
	err := m.Connect(ConnectRequest{})
	if err == nil {
		t.Fatal("expected error for missing server")
	}
}

func TestConnectFailsWithMissingBinary(t *testing.T) {
	t.Parallel()

	m := NewManager("__rdos_missing_binary__", t.TempDir()+"/session.log")
	err := m.Connect(ConnectRequest{Server: "rdp.local"})
	if err == nil {
		t.Fatal("expected error for missing binary")
	}

	s := m.Snapshot()
	if s.State != StateError {
		t.Fatalf("snapshot state = %q, want %q", s.State, StateError)
	}
}

func TestDisconnectWithoutSessionIsNoop(t *testing.T) {
	t.Parallel()

	m := NewManager("xfreerdp3", t.TempDir()+"/session.log")
	if err := m.Disconnect(); err != nil {
		t.Fatalf("disconnect should be noop: %v", err)
	}
}

func TestNormalizeUsernameForRDP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		domain   string
		want     string
	}{
		{name: "plain local username", username: "nick", domain: "", want: ".\\nick"},
		{name: "domain provided keeps username", username: "nick", domain: "CORP", want: "nick"},
		{name: "qualified username kept", username: "CORP\\nick", domain: "", want: "CORP\\nick"},
		{name: "upn username kept", username: "nick@example.com", domain: "", want: "nick@example.com"},
		{name: "trim whitespace", username: "  nick  ", domain: "", want: ".\\nick"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeUsernameForRDP(tt.username, tt.domain)
			if got != tt.want {
				t.Fatalf("normalizeUsernameForRDP(%q, %q) = %q, want %q", tt.username, tt.domain, got, tt.want)
			}
		})
	}
}
