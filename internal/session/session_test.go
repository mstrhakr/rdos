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
