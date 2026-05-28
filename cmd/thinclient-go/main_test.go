package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mstrhakr/rdos/internal/session"
	"github.com/mstrhakr/rdos/internal/tcconfig"
)

func TestEnvOrDefault(t *testing.T) {
	t.Parallel()

	const key = "RDOS_TEST_ENV_OR_DEFAULT"
	_ = os.Unsetenv(key)
	if got := envOrDefault(key, "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	if err := os.Setenv(key, "  value  "); err != nil {
		t.Fatalf("set env: %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	if got := envOrDefault(key, "fallback"); got != "value" {
		t.Fatalf("expected trimmed value, got %q", got)
	}
}

func TestHandleConfigRejectsInvalidKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tcconfig")
	store := tcconfig.NewStore(configPath)
	if err := store.Save(map[string]string{"server": "rdp.local"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	a := &app{
		store:    store,
		sessions: session.NewManager("xfreerdp3", filepath.Join(dir, "session.log")),
		version:  "test",
		config: map[string]string{
			"server": "rdp.local",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config", strings.NewReader(`{"values":{"bad-key":"x"}}`))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	a.handleConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleConfigPersistsValidKeys(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tcconfig")
	store := tcconfig.NewStore(configPath)
	if err := store.Save(map[string]string{"server": "rdp.local"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	a := &app{
		store:    store,
		sessions: session.NewManager("xfreerdp3", filepath.Join(dir, "session.log")),
		version:  "test",
		config: map[string]string{
			"server": "rdp.local",
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config", strings.NewReader(`{"values":{"helpdesk":"desk"}}`))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	a.handleConfig(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	saved, err := store.Load()
	if err != nil {
		t.Fatalf("reload saved config: %v", err)
	}
	if got := saved["helpdesk"]; got != "desk" {
		t.Fatalf("saved helpdesk = %q, want %q", got, "desk")
	}
}

func TestLoopbackOnlyRejectsNonLoopback(t *testing.T) {
	t.Parallel()

	a := &app{}
	h := a.loopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestHandleHealthReturnsJSON(t *testing.T) {
	t.Parallel()

	a := &app{version: "test", store: tcconfig.NewStore("/tmp/tcconfig")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:4321"
	w := httptest.NewRecorder()
	a.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["service"] != "thinclient-go" {
		t.Fatalf("service mismatch: %v", payload["service"])
	}
}

func TestHandleHealthIncludesBootMode(t *testing.T) {
	t.Parallel()

	a := &app{version: "test", bootMode: "legacy", store: tcconfig.NewStore("/tmp/tcconfig")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:1111"
	w := httptest.NewRecorder()
	a.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["bootMode"] != "legacy" {
		t.Fatalf("bootMode mismatch: %v", payload["bootMode"])
	}
}

func TestValidateNetworkSettings(t *testing.T) {
	t.Parallel()

	if err := validateNetworkSettings(networkSettings{Mode: "dhcp"}); err != nil {
		t.Fatalf("dhcp should be valid: %v", err)
	}
	if err := validateNetworkSettings(networkSettings{Mode: "static", Interface: "eth0", Address: "192.168.1.10", Prefix: "24"}); err != nil {
		t.Fatalf("static should be valid: %v", err)
	}
	if err := validateNetworkSettings(networkSettings{Mode: "static", Prefix: "24"}); err == nil {
		t.Fatal("expected error when static address missing")
	}
	if err := validateNetworkSettings(networkSettings{Mode: "static", Address: "192.168.1.10", Prefix: "24"}); err == nil {
		t.Fatal("expected error when static interface missing")
	}
	if err := validateNetworkSettings(networkSettings{Mode: "invalid"}); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestHandleNetworkInterfacesGet(t *testing.T) {
	t.Parallel()

	a := &app{config: map[string]string{"network_interface": "eth0"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/network/interfaces", nil)
	req.RemoteAddr = "127.0.0.1:7777"
	w := httptest.NewRecorder()

	a.handleNetworkInterfaces(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload networkInterfacesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Interfaces == nil {
		t.Fatal("interfaces should be present")
	}
	if payload.Wireless == nil {
		t.Fatal("wireless should be present")
	}
}

func TestHandleNetworkGetUsesDefaults(t *testing.T) {
	t.Parallel()

	a := &app{config: map[string]string{}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/network", nil)
	req.RemoteAddr = "127.0.0.1:3333"
	w := httptest.NewRecorder()

	a.handleNetwork(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload networkSettings
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Mode != "dhcp" {
		t.Fatalf("mode = %q, want dhcp", payload.Mode)
	}
}

func TestHandleNetworkPostPersistsConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "tcconfig")
	store := tcconfig.NewStore(configPath)
	if err := store.Save(map[string]string{"network_mode": "dhcp"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	a := &app{
		store:    store,
		sessions: session.NewManager("xfreerdp3", filepath.Join(dir, "session.log")),
		version:  "test",
		config: map[string]string{
			"network_mode": "dhcp",
		},
	}

	body := `{"mode":"static","interface":"eth0","address":"10.10.0.5","prefix":"24","gateway":"10.10.0.1","dns":"1.1.1.1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/network", strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	a.handleNetwork(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	saved, err := store.Load()
	if err != nil {
		t.Fatalf("reload saved config: %v", err)
	}
	if saved["network_mode"] != "static" {
		t.Fatalf("network_mode = %q, want static", saved["network_mode"])
	}
	if saved["static_address"] != "10.10.0.5" {
		t.Fatalf("static_address = %q, want 10.10.0.5", saved["static_address"])
	}
}

func TestHandleNetworkPostRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	a := &app{config: map[string]string{}, store: tcconfig.NewStore(filepath.Join(t.TempDir(), "tcconfig"))}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/network", strings.NewReader(`{"mode":"static"}`))
	req.RemoteAddr = "127.0.0.1:4444"
	w := httptest.NewRecorder()

	a.handleNetwork(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestIsSafeInterfaceName(t *testing.T) {
	t.Parallel()

	valid := []string{"eth0", "enp0s3", "wlan0", "wg0", "enx12:34"}
	for _, candidate := range valid {
		if !isSafeInterfaceName(candidate) {
			t.Fatalf("expected valid interface name %q", candidate)
		}
	}

	invalid := []string{"", "eth0;rm", "wlan0 $(x)", "../../etc/passwd"}
	for _, candidate := range invalid {
		if isSafeInterfaceName(candidate) {
			t.Fatalf("expected invalid interface name %q", candidate)
		}
	}
}
