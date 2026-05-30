package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestOTAStatusFromConfigDefaults(t *testing.T) {
	t.Parallel()

	status := otaStatusFromConfig(map[string]string{})

	if !status.AutoCheckEnabled {
		t.Fatal("auto check should default to enabled")
	}
	if status.AutoCheckSchedule != "daily" {
		t.Fatalf("auto check schedule = %q, want daily", status.AutoCheckSchedule)
	}
	if !status.AutoUpdateEnabled {
		t.Fatal("auto update should default to enabled")
	}
	if status.AutoUpdateSchedule != "daily" {
		t.Fatalf("auto update schedule = %q, want daily", status.AutoUpdateSchedule)
	}
	if status.Channel != "stable" {
		t.Fatalf("channel = %q, want stable", status.Channel)
	}
	if status.PendingRecovery {
		t.Fatal("pending recovery should default to false")
	}
}

func TestHandleOTAApplyPolicyRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/apply-policy", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAApplyPolicy(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOTAApplyPolicyReturnsUnavailableWithoutSudo(t *testing.T) {
	oldLookPath := otaLookPath
	t.Cleanup(func() {
		otaLookPath = oldLookPath
	})
	otaLookPath = func(file string) (string, error) {
		return "", errors.New("missing")
	}

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/apply-policy", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAApplyPolicy(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleOTAApplyPolicyReturnsOKOnSuccess(t *testing.T) {
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 0, "timer reloaded")

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/apply-policy", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAApplyPolicy(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleOTAUSBScanReturnsImages(t *testing.T) {
	oldScan := otaScanUSBImages
	t.Cleanup(func() {
		otaScanUSBImages = oldScan
	})

	otaScanUSBImages = func() []otaUSBImage {
		return []otaUSBImage{{Path: "/mnt/u/rdos.raw.zst", Mount: "/mnt/u", Filename: "rdos.raw.zst", Size: 123}}
	}

	a := &app{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/usb", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUSBScan(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleOTAUSBImportRejectsUnknownPath(t *testing.T) {
	oldFind := otaFindUSBImage
	t.Cleanup(func() {
		otaFindUSBImage = oldFind
	})
	otaFindUSBImage = func(path string) (otaUSBImage, bool) {
		return otaUSBImage{}, false
	}

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/usb/import", strings.NewReader(`{"path":"/mnt/u/rdos.raw.zst"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUSBImport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOTAUSBImportAcceptedOnSuccess(t *testing.T) {
	oldFind := otaFindUSBImage
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaFindUSBImage = oldFind
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaFindUSBImage = func(path string) (otaUSBImage, bool) {
		return otaUSBImage{Path: path}, true
	}
	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 0, "usb ota staged")

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/usb/import", strings.NewReader(`{"path":"/mnt/u/rdos.raw.zst"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUSBImport(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}
}

func TestHandleOTAUSBEventReadsStateFile(t *testing.T) {
	oldPath := otaUSBEventFile
	t.Cleanup(func() {
		otaUSBEventFile = oldPath
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "ota-event.json")
	if err := os.WriteFile(path, []byte(`{"detected":true,"filename":"rdos.raw.zst"}`), 0o600); err != nil {
		t.Fatalf("write event: %v", err)
	}
	otaUSBEventFile = path

	a := &app{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/usb/event", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUSBEvent(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload otaUSBEvent
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if !payload.Detected {
		t.Fatal("detected should be true")
	}
}

func TestHandleOTAStatusReturnsConfigValues(t *testing.T) {
	t.Parallel()

	a := &app{config: map[string]string{
		"auto_update_enabled": "false",
		"ota_channel":         "beta",
		"maintenance_window":  "03:30",
	}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload otaStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.AutoUpdateEnabled {
		t.Fatal("auto update should reflect config value false")
	}
	if payload.Channel != "beta" {
		t.Fatalf("channel = %q, want beta", payload.Channel)
	}
	if payload.MaintenanceWindow != "03:30" {
		t.Fatalf("maintenance window = %q, want 03:30", payload.MaintenanceWindow)
	}
}

func TestHandleOTAStatusRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAStatus(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOTAReleasesRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/releases", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAReleases(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOTAReleasesReturnsResults(t *testing.T) {
	oldFetchReleases := otaFetchReleases
	t.Cleanup(func() {
		otaFetchReleases = oldFetchReleases
	})

	otaFetchReleases = func(channel string, limit int) ([]otaReleaseEntry, error) {
		if channel != "beta" {
			t.Fatalf("channel = %q, want beta", channel)
		}
		if limit != 5 {
			t.Fatalf("limit = %d, want 5", limit)
		}
		return []otaReleaseEntry{{Tag: "v1.2.3-rc.1", Name: "v1.2.3-rc.1", Prerelease: true}}, nil
	}

	a := &app{config: map[string]string{"ota_channel": "beta"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/releases?limit=5", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAReleases(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload otaReleasesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Channel != "beta" {
		t.Fatalf("channel = %q, want beta", payload.Channel)
	}
	if len(payload.Releases) != 1 || payload.Releases[0].Tag != "v1.2.3-rc.1" {
		t.Fatalf("unexpected releases: %+v", payload.Releases)
	}
}

func TestHandleOTACatalogReturnsEntries(t *testing.T) {
	oldFetchReleases := otaFetchReleases
	t.Cleanup(func() {
		otaFetchReleases = oldFetchReleases
	})

	otaFetchReleases = func(channel string, limit int) ([]otaReleaseEntry, error) {
		if channel != "beta" {
			t.Fatalf("channel = %q, want beta", channel)
		}
		if limit != otaMaxLimit {
			t.Fatalf("limit = %d, want %d", limit, otaMaxLimit)
		}
		return []otaReleaseEntry{
			{Tag: "v1.3.0", Name: "v1.3.0", PublishedAt: "2026-05-29T00:00:00Z", Prerelease: false},
			{Tag: "v1.2.9", Name: "v1.2.9", PublishedAt: "2026-05-25T00:00:00Z", Prerelease: false},
		}, nil
	}

	a := &app{config: map[string]string{"ota_channel": "beta"}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/catalog", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTACatalog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload otaCatalogResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Channel != "beta" {
		t.Fatalf("channel = %q, want beta", payload.Channel)
	}
	if payload.LatestTag != "v1.3.0" {
		t.Fatalf("latestTag = %q, want v1.3.0", payload.LatestTag)
	}
	if len(payload.Entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(payload.Entries))
	}
}

func TestHandleOTACheckRunsInBackground(t *testing.T) {
	oldFetchReleases := otaFetchReleases
	t.Cleanup(func() {
		otaFetchReleases = oldFetchReleases
	})

	otaFetchReleases = func(channel string, limit int) ([]otaReleaseEntry, error) {
		return []otaReleaseEntry{{Tag: "v1.3.0", Name: "v1.3.0", Prerelease: false}}, nil
	}

	a := &app{config: map[string]string{"ota_channel": "stable"}}

	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/ota/check", strings.NewReader(`{}`))
	postReq.RemoteAddr = "127.0.0.1:5555"
	postW := httptest.NewRecorder()
	a.handleOTACheck(postW, postReq)

	if postW.Code != http.StatusAccepted {
		t.Fatalf("post status = %d, want %d", postW.Code, http.StatusAccepted)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodGet, "/api/v1/ota/check", nil)
		getReq.RemoteAddr = "127.0.0.1:5555"
		getW := httptest.NewRecorder()
		a.handleOTACheck(getW, getReq)

		if getW.Code != http.StatusOK {
			t.Fatalf("get status = %d, want %d", getW.Code, http.StatusOK)
		}

		var payload otaCheckResponse
		if err := json.Unmarshal(getW.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}

		if !payload.Running {
			if payload.LatestTag != "v1.3.0" {
				t.Fatalf("latestTag = %q, want v1.3.0", payload.LatestTag)
			}
			if !payload.Available {
				t.Fatal("available should be true")
			}
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("ota check did not complete before deadline")
}

func TestOTAReleaseMatchesChannelBetaIncludesStable(t *testing.T) {
	t.Parallel()

	betaStable := otaReleaseMatchesChannel(githubRelease{TagName: "v1.2.3", Prerelease: false}, "beta")
	if !betaStable {
		t.Fatal("beta channel should include stable releases")
	}

	stableRC := otaReleaseMatchesChannel(githubRelease{TagName: "v1.2.3-rc.1", Prerelease: true}, "stable")
	if stableRC {
		t.Fatal("stable channel should exclude rc/prerelease")
	}
}

func TestBuildOTACatalogEntriesLabels(t *testing.T) {
	t.Parallel()

	entries, latestTag := buildOTACatalogEntries([]otaReleaseEntry{
		{Tag: "v1.2.4", Name: "v1.2.4", Prerelease: false},
		{Tag: "v1.2.3-rc.1", Name: "v1.2.3-rc.1", Prerelease: true},
	}, "1.2.4")

	if latestTag != "v1.2.4" {
		t.Fatalf("latestTag = %q, want v1.2.4", latestTag)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}
	if !strings.Contains(strings.Join(entries[0].Labels, ","), "installed") {
		t.Fatalf("expected installed label on first entry: %+v", entries[0].Labels)
	}
	if !strings.Contains(strings.Join(entries[1].Labels, ","), "beta") {
		t.Fatalf("expected beta label on second entry: %+v", entries[1].Labels)
	}
}

func TestHandleOTAUpdateRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/update", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUpdate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOTAUpdateRejectsInvalidTag(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/update", strings.NewReader(`{"tag":"../bad"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleOTAUpdateReturnsUnavailableWithoutSudo(t *testing.T) {
	oldLookPath := otaLookPath
	t.Cleanup(func() {
		otaLookPath = oldLookPath
	})
	otaLookPath = func(file string) (string, error) {
		return "", errors.New("missing")
	}

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/update", strings.NewReader(`{"tag":"v1.2.3"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUpdate(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleOTAUpdateReturnsAcceptedOnSuccess(t *testing.T) {
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 0, "ota staging complete")

	a := &app{config: map[string]string{"ota_channel": "stable"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/update", strings.NewReader(`{"tag":"v1.2.3"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUpdate(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var payload otaUpdateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload.Tag != "v1.2.3" {
		t.Fatalf("tag = %q, want v1.2.3", payload.Tag)
	}
	if payload.Message != "ota staging complete" {
		t.Fatalf("message = %q, want ota staging complete", payload.Message)
	}
}

func TestHandleOTAUpdateReturnsBadRequestOnFailure(t *testing.T) {
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 1, "manifest channel mismatch")

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/update", strings.NewReader(`{"tag":"v1.2.3"}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTAUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "manifest channel mismatch") {
		t.Fatalf("body = %q, want update failure detail", w.Body.String())
	}
}

func TestHandleOTARollbackRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	a := &app{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/rollback", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTARollback(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleOTARollbackReturnsUnavailableWithoutSudo(t *testing.T) {
	oldLookPath := otaLookPath
	t.Cleanup(func() {
		otaLookPath = oldLookPath
	})
	otaLookPath = func(file string) (string, error) {
		return "", errors.New("missing")
	}

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/rollback", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTARollback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestHandleOTARollbackReturnsAcceptedOnSuccess(t *testing.T) {
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 0, "rollback queued")

	a := &app{config: map[string]string{"ota_channel": "beta"}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/rollback", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTARollback(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["message"] != "rollback queued" {
		t.Fatalf("message = %v, want rollback queued", payload["message"])
	}
	status, ok := payload["status"].(map[string]any)
	if !ok {
		t.Fatalf("status payload missing: %v", payload["status"])
	}
	if status["channel"] != "beta" {
		t.Fatalf("channel = %v, want beta", status["channel"])
	}
}

func TestHandleOTARollbackReturnsBadRequestOnFailure(t *testing.T) {
	oldLookPath := otaLookPath
	oldExecCommand := otaExecCommand
	t.Cleanup(func() {
		otaLookPath = oldLookPath
		otaExecCommand = oldExecCommand
	})

	otaLookPath = func(file string) (string, error) {
		return "/usr/bin/sudo", nil
	}
	otaExecCommand = fakeExecCommand(t, 1, "nothing to roll back to")

	a := &app{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/rollback", strings.NewReader(`{}`))
	req.RemoteAddr = "127.0.0.1:5555"
	w := httptest.NewRecorder()

	a.handleOTARollback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "nothing to roll back to") {
		t.Fatalf("body = %q, want rollback failure detail", w.Body.String())
	}
}

func fakeExecCommand(t *testing.T, exitCode int, stdout string) func(string, ...string) *exec.Cmd {
	t.Helper()

	return func(name string, args ...string) *exec.Cmd {
		cmdArgs := []string{"-test.run=TestHelperProcess", "--", name}
		cmdArgs = append(cmdArgs, args...)
		cmd := exec.Command(os.Args[0], cmdArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_EXIT_CODE="+strconv.Itoa(exitCode),
			"GO_HELPER_STDOUT="+stdout,
		)
		return cmd
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	_, _ = os.Stdout.WriteString(os.Getenv("GO_HELPER_STDOUT"))
	exitCode, err := strconv.Atoi(os.Getenv("GO_HELPER_EXIT_CODE"))
	if err != nil {
		exitCode = 0
	}
	os.Exit(exitCode)
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
