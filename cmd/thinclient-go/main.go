package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mstrhakr/rdos/internal/bootmode"
	"github.com/mstrhakr/rdos/internal/session"
	"github.com/mstrhakr/rdos/internal/tcconfig"
)

//go:embed web/*
var embeddedWeb embed.FS

type configPatch struct {
	Values map[string]string `json:"values"`
}

type networkSettings struct {
	Mode      string `json:"mode"`
	Interface string `json:"interface"`
	Address   string `json:"address"`
	Prefix    string `json:"prefix"`
	Gateway   string `json:"gateway"`
	DNS       string `json:"dns"`
}

type networkSettingsResponse struct {
	networkSettings
	ApplyMessage string `json:"applyMessage,omitempty"`
}

type otaStatusResponse struct {
	AutoUpdateEnabled bool   `json:"autoUpdateEnabled"`
	Channel           string `json:"channel"`
	MaintenanceWindow string `json:"maintenanceWindow"`
	CurrentSlot       string `json:"currentSlot"`
	PreviousSlot      string `json:"previousSlot"`
	BootTries         string `json:"bootTries"`
	PendingRecovery   bool   `json:"pendingRecovery"`
	CurrentVersion    string `json:"currentVersion"`
	InactiveVersion   string `json:"inactiveVersion"`
	GrubenvPath       string `json:"grubenvPath"`
	CanRollback       bool   `json:"canRollback"`
}

type otaReleaseEntry struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	PublishedAt string `json:"publishedAt"`
	Prerelease  bool   `json:"prerelease"`
}

type otaReleasesResponse struct {
	Channel  string            `json:"channel"`
	Releases []otaReleaseEntry `json:"releases"`
}

type otaCatalogEntry struct {
	Tag         string   `json:"tag"`
	Name        string   `json:"name"`
	PublishedAt string   `json:"publishedAt"`
	Prerelease  bool     `json:"prerelease"`
	Labels      []string `json:"labels"`
}

type otaCatalogResponse struct {
	Channel        string            `json:"channel"`
	CurrentVersion string            `json:"currentVersion"`
	LatestTag      string            `json:"latestTag,omitempty"`
	Entries        []otaCatalogEntry `json:"entries"`
}

type otaCheckResponse struct {
	Running        bool              `json:"running"`
	CheckedAt      string            `json:"checkedAt,omitempty"`
	Channel        string            `json:"channel,omitempty"`
	CurrentVersion string            `json:"currentVersion,omitempty"`
	LatestTag      string            `json:"latestTag,omitempty"`
	LatestName     string            `json:"latestName,omitempty"`
	Available      bool              `json:"available"`
	Error          string            `json:"error,omitempty"`
	Entries        []otaCatalogEntry `json:"entries,omitempty"`
}

type otaUpdateRequest struct {
	Tag string `json:"tag"`
}

type otaUpdateResponse struct {
	Message string            `json:"message"`
	Tag     string            `json:"tag,omitempty"`
	Status  otaStatusResponse `json:"status"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName     string               `json:"tag_name"`
	Name        string               `json:"name"`
	Draft       bool                 `json:"draft"`
	Prerelease  bool                 `json:"prerelease"`
	PublishedAt string               `json:"published_at"`
	Assets      []githubReleaseAsset `json:"assets"`
}

const (
	otaGitHubReleasesAPI = "https://api.github.com/repos/mstrhakr/rdos/releases"
	otaDefaultLimit      = 10
	otaMaxLimit          = 20
	terminalExecTimeout  = 15 * time.Second
	terminalOutputMax    = 64 * 1024
	terminalCommandMax   = 2048
	ttydPort             = 7681
)

type networkInterfaceInfo struct {
	Name       string   `json:"name"`
	Operstate  string   `json:"operstate"`
	IsWireless bool     `json:"isWireless"`
	Addresses  []string `json:"addresses"`
	MAC        string   `json:"mac"`
	SSID       string   `json:"ssid,omitempty"`
}

type networkInterfacesResponse struct {
	Interfaces       []string               `json:"interfaces"`
	Wireless         []string               `json:"wireless"`
	HasWireless      bool                   `json:"hasWireless"`
	DefaultInterface string                 `json:"defaultInterface"`
	DefaultWireless  string                 `json:"defaultWireless"`
	Details          []networkInterfaceInfo `json:"details"`
}

type wireguardUSBConfig struct {
	Path        string `json:"path"`
	Mount       string `json:"mount"`
	Filename    string `json:"filename"`
	Interface   string `json:"interface"`
	NeedsImport bool   `json:"needsImport"`
}

type wireguardUSBScanResponse struct {
	Configs []wireguardUSBConfig `json:"configs"`
}

type wireguardUSBImportRequest struct {
	Path string `json:"path"`
}

type wireguardUSBImportResponse struct {
	Path      string `json:"path"`
	Interface string `json:"interface"`
	Message   string `json:"message"`
}

type wifiNetwork struct {
	SSID      string `json:"ssid"`
	Signal    string `json:"signal"`
	Security  string `json:"security"`
	Interface string `json:"interface"`
}

type wifiConnectRequest struct {
	Interface string `json:"interface"`
	SSID      string `json:"ssid"`
	Password  string `json:"password"`
	Hidden    bool   `json:"hidden"`
}

type wifiConnectState struct {
	Interface string `json:"interface"`
	SSID      string `json:"ssid"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

type terminalExecRequest struct {
	Command string `json:"command"`
}

type terminalExecResponse struct {
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	Output     string `json:"output"`
	DurationMs int64  `json:"durationMs"`
	TimedOut   bool   `json:"timedOut"`
}

type terminalTTYDResponse struct {
	Running bool   `json:"running"`
	Ready   bool   `json:"ready"`
	URL     string `json:"url"`
	Message string `json:"message"`
}

type statusSnapshot struct {
	Time          string `json:"time"`
	BootMode      string `json:"bootMode"`
	Hostname      string `json:"hostname"`
	IP            string `json:"ip"`
	WiFi          string `json:"wifi"`
	WireGuard     string `json:"wireguard"`
	Battery       string `json:"battery"`
	Overlay       string `json:"overlay"`
	Wallpaper     string `json:"wallpaper"`
	Helpdesk      string `json:"helpdesk"`
	Server        string `json:"server"`
	Connection    string `json:"connection"`
	StatusEnabled string `json:"statusEnabled"`
	WiFiInterface string `json:"wifiInterface"`
}

type app struct {
	store    *tcconfig.Store
	sessions *session.Manager
	version  string
	bootMode string

	mu         sync.Mutex
	config     map[string]string
	terminalMu sync.Mutex
	ttydCmd    *exec.Cmd
	otaCheckMu sync.Mutex
	otaCheck   otaCheckResponse
}

var otaLookPath = exec.LookPath
var otaExecCommand = exec.Command
var otaFetchReleases = fetchOTAReleases
var otaHTTPClient = &http.Client{Timeout: 20 * time.Second}

func main() {
	listenAddr := flag.String("listen", envOrDefault("RDOS_UI_LISTEN", "127.0.0.1:8080"), "web server listen address")
	tcconfigPath := flag.String("tcconfig", envOrDefault("RDOS_TCCONFIG", "/home/thinclient/tcconfig"), "path to tcconfig")
	rdpBinary := flag.String("rdp-binary", envOrDefault("RDOS_RDP_BINARY", "xfreerdp3"), "RDP binary path")
	sessionLog := flag.String("session-log", envOrDefault("RDOS_SESSION_LOG", filepath.Join(os.TempDir(), "rdos-session.log")), "session log path")
	flag.Parse()

	cfgStore := tcconfig.NewStore(*tcconfigPath)
	cfg, err := cfgStore.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	uiFS, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		log.Fatalf("open embedded web assets: %v", err)
	}

	application := &app{
		store:    cfgStore,
		sessions: session.NewManager(*rdpBinary, *sessionLog),
		version:  readVersion(),
		bootMode: detectBootMode(),
		config:   cfg,
	}

	// Keep a recovery console available by default when the UI is active.
	go func() {
		statusCode, status := application.startTerminalTTYD()
		if statusCode >= http.StatusBadRequest {
			log.Printf("terminal startup skipped: %s", status.Message)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/api/v1/health", application.loopbackOnly(http.HandlerFunc(application.handleHealth)))
	mux.Handle("/api/v1/config", application.loopbackOnly(http.HandlerFunc(application.handleConfig)))
	mux.Handle("/api/v1/network", application.loopbackOnly(http.HandlerFunc(application.handleNetwork)))
	mux.Handle("/api/v1/network/interfaces", application.loopbackOnly(http.HandlerFunc(application.handleNetworkInterfaces)))
	mux.Handle("/api/v1/ota", application.loopbackOnly(http.HandlerFunc(application.handleOTAStatus)))
	mux.Handle("/api/v1/ota/releases", application.loopbackOnly(http.HandlerFunc(application.handleOTAReleases)))
	mux.Handle("/api/v1/ota/catalog", application.loopbackOnly(http.HandlerFunc(application.handleOTACatalog)))
	mux.Handle("/api/v1/ota/check", application.loopbackOnly(http.HandlerFunc(application.handleOTACheck)))
	mux.Handle("/api/v1/ota/update", application.loopbackOnly(http.HandlerFunc(application.handleOTAUpdate)))
	mux.Handle("/api/v1/ota/rollback", application.loopbackOnly(http.HandlerFunc(application.handleOTARollback)))
	mux.Handle("/api/v1/wireguard/usb", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardUSBScan)))
	mux.Handle("/api/v1/wireguard/import", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardUSBImport)))
	mux.Handle("/api/v1/status", application.loopbackOnly(http.HandlerFunc(application.handleStatus)))
	mux.Handle("/api/v1/wifi/scan", application.loopbackOnly(http.HandlerFunc(application.handleWifiScan)))
	mux.Handle("/api/v1/wifi/connect", application.loopbackOnly(http.HandlerFunc(application.handleWifiConnect)))
	mux.Handle("/api/v1/terminal/exec", application.loopbackOnly(http.HandlerFunc(application.handleTerminalExec)))
	mux.Handle("/api/v1/terminal/ttyd", application.loopbackOnly(http.HandlerFunc(application.handleTerminalTTYD)))
	mux.Handle("/api/v1/session", application.loopbackOnly(http.HandlerFunc(application.handleSessionStatus)))
	mux.Handle("/api/v1/session/connect", application.loopbackOnly(http.HandlerFunc(application.handleSessionConnect)))
	mux.Handle("/api/v1/session/disconnect", application.loopbackOnly(http.HandlerFunc(application.handleSessionDisconnect)))
	mux.Handle("/", http.FileServer(http.FS(uiFS)))

	httpServer := &http.Server{
		Addr:              *listenAddr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("thinclient-go listening on %s", *listenAddr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http server failed: %v", err)
	}
}

func (a *app) loopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if host != "127.0.0.1" && host != "::1" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"service":    "thinclient-go",
		"version":    a.version,
		"bootMode":   a.bootMode,
		"configPath": a.store.Path(),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
}

func (a *app) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.Lock()
		cfg := cloneConfig(a.config)
		a.mu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"values": cfg})
		return
	case http.MethodPost:
		defer r.Body.Close()
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		var patch configPatch
		if err := json.Unmarshal(body, &patch); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}

		a.mu.Lock()
		updated := cloneConfig(a.config)
		for key, value := range patch.Values {
			normalizedKey := strings.TrimSpace(key)
			if !tcconfig.ValidKey(normalizedKey) {
				http.Error(w, "invalid config key: "+normalizedKey, http.StatusBadRequest)
				a.mu.Unlock()
				return
			}
			updated[normalizedKey] = strings.TrimSpace(value)
		}
		a.mu.Unlock()

		if err := a.store.Save(updated); err != nil {
			http.Error(w, "failed to persist config", http.StatusInternalServerError)
			return
		}

		a.mu.Lock()
		a.config = updated
		a.mu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"values": cloneConfig(updated)})
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snapshot := a.sessions.Snapshot()
	respondJSON(w, http.StatusOK, snapshot)
}

func (a *app) handleNetwork(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.mu.Lock()
		settings := networkSettingsFromConfig(a.config)
		a.mu.Unlock()
		respondJSON(w, http.StatusOK, settings)
		return
	case http.MethodPost:
		defer r.Body.Close()

		var settings networkSettings
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&settings); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}

		if err := validateNetworkSettings(settings); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		a.mu.Lock()
		updated := cloneConfig(a.config)
		applyNetworkSettings(updated, settings)
		a.mu.Unlock()

		if err := a.store.Save(updated); err != nil {
			http.Error(w, "failed to persist config", http.StatusInternalServerError)
			return
		}

		a.mu.Lock()
		a.config = updated
		a.mu.Unlock()

		applyMessage := applyNetworkFromConfig(a.store.Path())
		respondJSON(w, http.StatusOK, networkSettingsResponse{
			networkSettings: networkSettingsFromConfig(updated),
			ApplyMessage:    applyMessage,
		})
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	respondJSON(w, http.StatusOK, statusSnapshot{
		Time:          time.Now().Format(time.RFC3339),
		BootMode:      a.bootMode,
		Hostname:      hostnameValue(),
		IP:            primaryIP(),
		WiFi:          wifiStatusText(),
		WireGuard:     wireguardStatusText(),
		Battery:       batteryStatusText(),
		Overlay:       boolText(cfg["status_overlay_enabled"], "on", "off"),
		Wallpaper:     strings.TrimSpace(cfg["wallpaper_mode"]),
		Helpdesk:      strings.TrimSpace(cfg["helpdesk"]),
		Server:        strings.TrimSpace(cfg["server"]),
		Connection:    strings.TrimSpace(cfg["network_mode"]),
		StatusEnabled: strings.TrimSpace(cfg["status_overlay_enabled"]),
		WiFiInterface: strings.TrimSpace(cfg["network_interface"]),
	})
}

func (a *app) handleOTAStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	respondJSON(w, http.StatusOK, otaStatusFromConfig(cfg))
}

func (a *app) handleOTAReleases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := otaDefaultLimit
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit < 1 || parsedLimit > otaMaxLimit {
			http.Error(w, "limit must be between 1 and 20", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	channel := otaChannelFromConfig(cfg)
	releases, err := otaFetchReleases(channel, limit)
	if err != nil {
		http.Error(w, "failed to fetch releases: "+err.Error(), http.StatusBadGateway)
		return
	}

	respondJSON(w, http.StatusOK, otaReleasesResponse{
		Channel:  channel,
		Releases: releases,
	})
}

func (a *app) handleOTAUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req otaUpdateRequest
	if strings.TrimSpace(string(body)) != "" {
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "invalid json payload", http.StatusBadRequest)
			return
		}
	}

	tag := strings.TrimSpace(req.Tag)
	if tag != "" && !isValidOTATag(tag) {
		http.Error(w, "invalid tag", http.StatusBadRequest)
		return
	}

	if _, err := otaLookPath("sudo"); err != nil {
		http.Error(w, "update unavailable: sudo not found", http.StatusInternalServerError)
		return
	}

	args := []string{"-n", "/usr/bin/tc-ota-updater", "--manual"}
	if tag != "" {
		args = append(args, "--tag", tag)
	}

	cmd := otaExecCommand("sudo", args...)
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		http.Error(w, "update failed: "+message, http.StatusBadRequest)
		return
	}
	if message == "" {
		message = "OTA update started"
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	respondJSON(w, http.StatusAccepted, otaUpdateResponse{
		Message: message,
		Tag:     tag,
		Status:  otaStatusFromConfig(cfg),
	})
}

func (a *app) handleOTACatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	channel := otaChannelFromConfig(cfg)
	status := otaStatusFromConfig(cfg)
	releases, err := otaFetchReleases(channel, otaMaxLimit)
	if err != nil {
		http.Error(w, "failed to fetch catalog: "+err.Error(), http.StatusBadGateway)
		return
	}

	entries, latestTag := buildOTACatalogEntries(releases, status.CurrentVersion)
	respondJSON(w, http.StatusOK, otaCatalogResponse{
		Channel:        channel,
		CurrentVersion: status.CurrentVersion,
		LatestTag:      latestTag,
		Entries:        entries,
	})
}

func (a *app) handleOTACheck(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.otaCheckMu.Lock()
		snapshot := a.otaCheck
		a.otaCheckMu.Unlock()
		respondJSON(w, http.StatusOK, snapshot)
		return
	case http.MethodPost:
		a.otaCheckMu.Lock()
		if a.otaCheck.Running {
			snapshot := a.otaCheck
			a.otaCheckMu.Unlock()
			respondJSON(w, http.StatusAccepted, snapshot)
			return
		}
		a.otaCheck.Running = true
		a.otaCheck.Error = ""
		a.otaCheckMu.Unlock()

		go a.runOTACheck()

		a.otaCheckMu.Lock()
		snapshot := a.otaCheck
		a.otaCheckMu.Unlock()
		respondJSON(w, http.StatusAccepted, snapshot)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) runOTACheck() {
	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	channel := otaChannelFromConfig(cfg)
	status := otaStatusFromConfig(cfg)
	result := otaCheckResponse{
		Running:        false,
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		Channel:        channel,
		CurrentVersion: status.CurrentVersion,
	}

	releases, err := otaFetchReleases(channel, otaMaxLimit)
	if err != nil {
		result.Error = "failed to fetch releases: " + err.Error()
		a.otaCheckMu.Lock()
		a.otaCheck = result
		a.otaCheckMu.Unlock()
		return
	}

	entries, latestTag := buildOTACatalogEntries(releases, status.CurrentVersion)
	result.Entries = entries
	result.LatestTag = latestTag
	if len(entries) > 0 {
		result.LatestName = entries[0].Name
	}
	result.Available = latestTag != "" && !otaTagMatchesCurrentVersion(latestTag, status.CurrentVersion)

	a.otaCheckMu.Lock()
	a.otaCheck = result
	a.otaCheckMu.Unlock()
}

func (a *app) handleOTARollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, err := otaLookPath("sudo"); err != nil {
		http.Error(w, "rollback unavailable: sudo not found", http.StatusInternalServerError)
		return
	}

	cmd := otaExecCommand("sudo", "-n", "/usr/bin/tc-ota-rollback")
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		http.Error(w, "rollback failed: "+message, http.StatusBadRequest)
		return
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	respondJSON(w, http.StatusAccepted, map[string]any{
		"message": message,
		"status":  otaStatusFromConfig(cfg),
	})
}

func (a *app) handleNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaces, wireless := listNetworkInterfaces()

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	defaultInterface := strings.TrimSpace(cfg["network_interface"])
	defaultWireless := ""
	for _, iface := range wireless {
		if iface == defaultInterface {
			defaultWireless = iface
			break
		}
	}
	if defaultWireless == "" && len(wireless) > 0 {
		defaultWireless = wireless[0]
	}

	respondJSON(w, http.StatusOK, networkInterfacesResponse{
		Interfaces:       interfaces,
		Wireless:         wireless,
		HasWireless:      len(wireless) > 0,
		DefaultInterface: defaultInterface,
		DefaultWireless:  defaultWireless,
		Details:          networkInterfaceDetails(interfaces, wireless),
	})
}

func (a *app) handleWireGuardUSBScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	respondJSON(w, http.StatusOK, wireguardUSBScanResponse{Configs: scanWireGuardUSBConfigs()})
}

func (a *app) handleWireGuardUSBImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req wireguardUSBImportRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	config, ok := findWireGuardUSBConfig(path)
	if !ok {
		http.Error(w, "wireguard config not found on a USB drive", http.StatusNotFound)
		return
	}

	cmd := exec.Command("sudo", "-n", "/usr/bin/tc-configure-wireguard", "--from-usb-drive", config.Path)
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		http.Error(w, "wireguard import failed: "+message, http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, wireguardUSBImportResponse{
		Path:      config.Path,
		Interface: config.Interface,
		Message:   message,
	})
}

func (a *app) handleTerminalExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req terminalExecRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	command := strings.TrimSpace(req.Command)
	if command == "" {
		http.Error(w, "command is required", http.StatusBadRequest)
		return
	}
	if len(command) > terminalCommandMax {
		http.Error(w, "command is too long", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), terminalExecTimeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	outputBytes, err := cmd.CombinedOutput()
	durationMs := time.Since(start).Milliseconds()

	timedOut := errors.Is(ctx.Err(), context.DeadlineExceeded)
	exitCode := 0
	if err != nil {
		if timedOut {
			exitCode = 124
		} else if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	if len(outputBytes) > terminalOutputMax {
		outputBytes = append(outputBytes[:terminalOutputMax], []byte("\n[output truncated]\n")...)
	}

	output := strings.TrimSpace(string(outputBytes))
	if output == "" && err != nil {
		output = err.Error()
	}
	if output == "" {
		output = "(no output)"
	}

	respondJSON(w, http.StatusOK, terminalExecResponse{
		Command:    command,
		ExitCode:   exitCode,
		Output:     output,
		DurationMs: durationMs,
		TimedOut:   timedOut,
	})
}

func (a *app) handleTerminalTTYD(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, a.terminalTTYDStatus(""))
		return
	case http.MethodPost:
		statusCode, status := a.startTerminalTTYD()
		respondJSON(w, statusCode, status)
		return
	case http.MethodDelete:
		statusCode, status := a.stopTerminalTTYD()
		respondJSON(w, statusCode, status)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) terminalTTYDStatus(message string) terminalTTYDResponse {
	running := false
	a.terminalMu.Lock()
	running = a.ttydCmd != nil
	a.terminalMu.Unlock()

	ready := ttydReady()
	if !running {
		ready = false
	}

	if message == "" {
		if running {
			if ready {
				message = "Console running"
			} else {
				message = "Console starting"
			}
		} else {
			message = "Console stopped"
		}
	}

	return terminalTTYDResponse{
		Running: running,
		Ready:   ready,
		URL:     "http://127.0.0.1:" + strconv.Itoa(ttydPort) + "/",
		Message: message,
	}
}

func (a *app) startTerminalTTYD() (int, terminalTTYDResponse) {
	if _, err := exec.LookPath("ttyd"); err != nil {
		return http.StatusBadRequest, a.terminalTTYDStatus("ttyd is not installed in this image")
	}

	a.terminalMu.Lock()
	if a.ttydCmd != nil {
		a.terminalMu.Unlock()
		return http.StatusOK, a.terminalTTYDStatus("Console already running")
	}

	cmd := exec.Command("ttyd", "-W", "-p", strconv.Itoa(ttydPort), "bash", "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err := cmd.Start(); err != nil {
		a.terminalMu.Unlock()
		return http.StatusBadRequest, a.terminalTTYDStatus("Failed to start ttyd: " + err.Error())
	}
	a.ttydCmd = cmd
	a.terminalMu.Unlock()

	go func(started *exec.Cmd) {
		_ = started.Wait()
		a.terminalMu.Lock()
		if a.ttydCmd == started {
			a.ttydCmd = nil
		}
		a.terminalMu.Unlock()
	}(cmd)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ttydReady() {
			return http.StatusOK, a.terminalTTYDStatus("Console running")
		}
		time.Sleep(120 * time.Millisecond)
	}

	return http.StatusAccepted, a.terminalTTYDStatus("Console started, waiting for readiness")
}

func (a *app) stopTerminalTTYD() (int, terminalTTYDResponse) {
	a.terminalMu.Lock()
	cmd := a.ttydCmd
	a.ttydCmd = nil
	a.terminalMu.Unlock()

	if cmd == nil {
		return http.StatusOK, a.terminalTTYDStatus("Console already stopped")
	}

	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	return http.StatusOK, a.terminalTTYDStatus("Console stopped")
}

func ttydReady() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+strconv.Itoa(ttydPort), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (a *app) handleWifiScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	iface := strings.TrimSpace(r.URL.Query().Get("interface"))
	if iface != "" {
		if !isSafeInterfaceName(iface) {
			http.Error(w, "invalid interface", http.StatusBadRequest)
			return
		}
	}

	args := []string{"-n", "/usr/bin/tc-scan-wifi"}
	if iface != "" {
		args = append(args, iface)
	}

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "wifi scan failed: "+strings.TrimSpace(string(output)), http.StatusBadRequest)
		return
	}

	networks := parseWifiScanOutput(string(output), iface)
	respondJSON(w, http.StatusOK, map[string]any{
		"interface": iface,
		"networks":  networks,
	})
}

func (a *app) handleWifiConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req wifiConnectRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Interface) == "" {
		http.Error(w, "interface is required", http.StatusBadRequest)
		return
	}
	if !isSafeInterfaceName(strings.TrimSpace(req.Interface)) {
		http.Error(w, "invalid interface", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.SSID) == "" {
		http.Error(w, "ssid is required", http.StatusBadRequest)
		return
	}

	requestPath := filepath.Join(os.TempDir(), "rdos-wifi-request.txt")
	payload := strings.Builder{}
	payload.WriteString("interface=")
	payload.WriteString(req.Interface)
	payload.WriteString("\nssid=")
	payload.WriteString(req.SSID)
	payload.WriteString("\nhidden=")
	payload.WriteString(strconv.FormatBool(req.Hidden))
	payload.WriteString("\npassword=")
	payload.WriteString(req.Password)
	payload.WriteString("\n")
	if err := os.WriteFile(requestPath, []byte(payload.String()), 0o600); err != nil {
		http.Error(w, "failed to write wifi request", http.StatusInternalServerError)
		return
	}
	defer func() { _ = os.Remove(requestPath) }()

	cmd := exec.Command("sudo", "-n", "/usr/bin/tc-apply-wifi-request")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "wifi connect failed: "+strings.TrimSpace(string(output)), http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusAccepted, wifiConnectState{
		Interface: req.Interface,
		SSID:      req.SSID,
		Status:    "applied",
		Message:   strings.TrimSpace(string(output)),
	})
}

func (a *app) handleSessionConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req session.ConnectRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	// Stop the embedded terminal while an RDP session is active.
	_, _ = a.stopTerminalTTYD()

	if err := a.sessions.Connect(req); err != nil {
		// Restore terminal access when RDP launch fails.
		_, _ = a.startTerminalTTYD()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	respondJSON(w, http.StatusAccepted, a.sessions.Snapshot())
}

func (a *app) handleSessionDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := a.sessions.Disconnect(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = a.startTerminalTTYD()
	respondJSON(w, http.StatusOK, a.sessions.Snapshot())
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func cloneConfig(cfg map[string]string) map[string]string {
	cloned := make(map[string]string, len(cfg))
	for k, v := range cfg {
		cloned[k] = v
	}
	return cloned
}

func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func readVersion() string {
	b, err := os.ReadFile("/tcversion")
	if err != nil {
		return "dev"
	}
	v := strings.TrimSpace(string(b))
	if v == "" {
		return "dev"
	}
	return v
}

func hostnameValue() string {
	host, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return host
}

func primaryIP() string {
	routeOutput, err := exec.Command("ip", "-4", "route", "get", "1.1.1.1").CombinedOutput()
	if err == nil {
		fields := strings.Fields(string(routeOutput))
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "src" {
				return fields[i+1]
			}
		}
	}

	output, err := exec.Command("hostname", "--all-ip-addresses").CombinedOutput()
	if err != nil {
		return "none"
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 {
		return "none"
	}
	return fields[0]
}

func wifiStatusText() string {
	for _, iface := range wirelessInterfaces() {
		operstate, _ := os.ReadFile(filepath.Join("/sys/class/net", iface, "operstate"))
		if strings.TrimSpace(string(operstate)) != "up" {
			continue
		}
		if ssid := wifiSSID(iface); ssid != "" {
			return iface + ":" + ssid
		}
		return iface + ":connected"
	}
	return "WiFi n/a"
}

func wireguardStatusText() string {
	if _, err := exec.LookPath("wg"); err != nil {
		return "WG n/a"
	}
	output, err := exec.Command("wg", "show", "interfaces").CombinedOutput()
	if err != nil {
		return "WG down"
	}
	interfaces := strings.Fields(strings.TrimSpace(string(output)))
	if len(interfaces) == 0 {
		return "WG down"
	}
	return "WG up(" + strings.Join(interfaces, ",") + ")"
}

func batteryStatusText() string {
	for _, device := range powerSupplyDevices() {
		typePath := filepath.Join(device, "type")
		capacityPath := filepath.Join(device, "capacity")
		kind, err := os.ReadFile(typePath)
		if err != nil || strings.TrimSpace(string(kind)) != "Battery" {
			continue
		}
		capacity, err := os.ReadFile(capacityPath)
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(capacity)) + "%"
	}
	return "n/a"
}

func wirelessInterfaces() []string {
	devices, _ := filepath.Glob("/sys/class/net/*")
	ifaces := make([]string, 0, len(devices))
	for _, device := range devices {
		if _, err := os.Stat(filepath.Join(device, "wireless")); err == nil {
			ifaces = append(ifaces, filepath.Base(device))
		}
	}
	sort.Strings(ifaces)
	return ifaces
}

func wifiSSID(iface string) string {
	if _, err := exec.LookPath("iw"); err == nil {
		output, err := exec.Command("iw", "dev", iface, "link").CombinedOutput()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "SSID: ") {
					return strings.TrimSpace(strings.TrimPrefix(line, "SSID: "))
				}
			}
		}
	}
	return ""
}

func powerSupplyDevices() []string {
	devices, _ := filepath.Glob("/sys/class/power_supply/*")
	sort.Strings(devices)
	return devices
}

func boolText(value string, trueText, falseText string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return trueText
	default:
		return falseText
	}
}

func isSafeInterfaceName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func parseWifiScanOutput(output string, iface string) []wifiNetwork {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	networksBySSID := map[string]wifiNetwork{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		ssid := strings.TrimSpace(parts[0])
		signal := strings.TrimSpace(parts[1])
		security := strings.TrimSpace(parts[2])
		if ssid == "" {
			continue
		}
		networksBySSID[ssid] = wifiNetwork{SSID: ssid, Signal: signal, Security: security, Interface: iface}
	}
	networks := make([]wifiNetwork, 0, len(networksBySSID))
	for _, network := range networksBySSID {
		networks = append(networks, network)
	}
	sort.Slice(networks, func(i, j int) bool { return networks[i].SSID < networks[j].SSID })
	return networks
}

func detectBootMode() string {
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return bootmode.ModeWeb
	}
	return bootmode.Resolve(string(cmdline), bootmode.ModeWeb)
}

func otaStatusFromConfig(cfg map[string]string) otaStatusResponse {
	grubenvPath := resolveGrubenvPath()
	grubenvValues := map[string]string{}
	if grubenvPath != "" {
		grubenvValues = readGrubEnvValues(grubenvPath)
	}

	currentSlot := strings.TrimSpace(grubenvValues["current_slot"])
	previousSlot := strings.TrimSpace(grubenvValues["previous_slot"])
	bootTries := strings.TrimSpace(grubenvValues["boot_tries"])
	pendingRecovery := boolValue(grubenvValues["pending_recovery"], false)
	inactiveSlot := ""
	switch currentSlot {
	case "a":
		inactiveSlot = "b"
	case "b":
		inactiveSlot = "a"
	}

	currentVersion := readSlotVersion(currentSlot)
	if currentVersion == "" {
		currentVersion = readVersion()
	}
	inactiveVersion := readSlotVersion(inactiveSlot)

	return otaStatusResponse{
		AutoUpdateEnabled: boolValue(cfg["auto_update_enabled"], true),
		Channel:           strings.TrimSpace(defaultString(cfg["ota_channel"], "stable")),
		MaintenanceWindow: strings.TrimSpace(cfg["maintenance_window"]),
		CurrentSlot:       currentSlot,
		PreviousSlot:      previousSlot,
		BootTries:         bootTries,
		PendingRecovery:   pendingRecovery,
		CurrentVersion:    currentVersion,
		InactiveVersion:   inactiveVersion,
		GrubenvPath:       grubenvPath,
		CanRollback:       currentSlot != "" && previousSlot != "" && currentSlot != previousSlot,
	}
}

func resolveGrubenvPath() string {
	for _, candidate := range []string{"/boot/grub/grubenv", "/boot/efi/grub/grubenv", "/efi/grub/grubenv"} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func readGrubEnvValues(path string) map[string]string {
	values := map[string]string{}
	if path == "" {
		return values
	}
	if _, err := exec.LookPath("grub-editenv"); err != nil {
		return values
	}

	output, err := exec.Command("grub-editenv", path, "list").CombinedOutput()
	if err != nil {
		return values
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.IndexRune(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		values[key] = value
	}

	return values
}

func readSlotVersion(slot string) string {
	slot = strings.TrimSpace(slot)
	if slot == "" {
		return ""
	}

	candidate := filepath.Join("/boot/slots", slot+"-version")
	if b, err := os.ReadFile(candidate); err == nil {
		return strings.TrimSpace(string(b))
	}

	return ""
}

func boolValue(value string, defaultValue bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return defaultValue
	default:
		return defaultValue
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func otaChannelFromConfig(cfg map[string]string) string {
	channel := strings.ToLower(strings.TrimSpace(defaultString(cfg["ota_channel"], "stable")))
	if channel == "beta" {
		return "beta"
	}
	return "stable"
}

func otaTagMatchesCurrentVersion(tag, current string) bool {
	tag = strings.TrimSpace(tag)
	current = strings.TrimSpace(current)
	if tag == "" || current == "" {
		return false
	}
	if tag == current {
		return true
	}
	if strings.HasPrefix(tag, "v") && strings.TrimPrefix(tag, "v") == current {
		return true
	}
	if strings.HasPrefix(current, "v") && strings.TrimPrefix(current, "v") == tag {
		return true
	}
	return false
}

func buildOTACatalogEntries(releases []otaReleaseEntry, currentVersion string) ([]otaCatalogEntry, string) {
	entries := make([]otaCatalogEntry, 0, len(releases))
	latestTag := ""
	for idx, release := range releases {
		labels := make([]string, 0, 4)
		if idx == 0 {
			labels = append(labels, "latest")
			latestTag = strings.TrimSpace(release.Tag)
		}
		if release.Prerelease || strings.Contains(strings.ToLower(release.Tag), "-rc.") {
			labels = append(labels, "beta")
		} else {
			labels = append(labels, "stable")
		}
		if otaTagMatchesCurrentVersion(release.Tag, currentVersion) {
			labels = append(labels, "installed")
		} else {
			labels = append(labels, "available")
		}

		entries = append(entries, otaCatalogEntry{
			Tag:         release.Tag,
			Name:        release.Name,
			PublishedAt: release.PublishedAt,
			Prerelease:  release.Prerelease,
			Labels:      labels,
		})
	}

	return entries, latestTag
}

func fetchOTAReleases(channel string, limit int) ([]otaReleaseEntry, error) {
	if limit < 1 {
		limit = otaDefaultLimit
	}
	if limit > otaMaxLimit {
		limit = otaMaxLimit
	}

	perPage := limit * 4
	if perPage < 20 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	req, err := http.NewRequest(http.MethodGet, otaGitHubReleasesAPI+"?per_page="+strconv.Itoa(perPage), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "rdos-thinclient-go")

	resp, err := otaHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, errors.New("github api returned " + resp.Status + ": " + strings.TrimSpace(string(body)))
	}

	var releases []githubRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&releases); err != nil {
		return nil, err
	}

	filtered := make([]otaReleaseEntry, 0, limit)
	for _, release := range releases {
		if len(filtered) >= limit {
			break
		}
		if release.Draft {
			continue
		}
		if !otaReleaseMatchesChannel(release, channel) {
			continue
		}
		if !otaReleaseHasRequiredAssets(release) {
			continue
		}

		name := strings.TrimSpace(release.Name)
		if name == "" {
			name = strings.TrimSpace(release.TagName)
		}

		filtered = append(filtered, otaReleaseEntry{
			Tag:         strings.TrimSpace(release.TagName),
			Name:        name,
			PublishedAt: strings.TrimSpace(release.PublishedAt),
			Prerelease:  release.Prerelease,
		})
	}

	return filtered, nil
}

func otaReleaseMatchesChannel(release githubRelease, channel string) bool {
	tag := strings.ToLower(strings.TrimSpace(release.TagName))
	isRC := strings.Contains(tag, "-rc.")
	if channel == "beta" {
		return true
	}
	return !release.Prerelease && !isRC
}

func otaReleaseHasRequiredAssets(release githubRelease) bool {
	hasManifest := false
	hasImage := false
	for _, asset := range release.Assets {
		switch strings.TrimSpace(asset.Name) {
		case "manifest.json":
			hasManifest = true
		case "rdos-prod.raw.zst":
			hasImage = true
		}
	}
	return hasManifest && hasImage
}

func isValidOTATag(tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" || !strings.HasPrefix(tag, "v") {
		return false
	}
	for _, r := range tag {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' || r == '+' {
			continue
		}
		return false
	}
	return true
}

func networkSettingsFromConfig(cfg map[string]string) networkSettings {
	mode := strings.TrimSpace(cfg["network_mode"])
	if mode == "" {
		mode = "dhcp"
	}

	return networkSettings{
		Mode:      mode,
		Interface: strings.TrimSpace(cfg["network_interface"]),
		Address:   strings.TrimSpace(cfg["static_address"]),
		Prefix:    strings.TrimSpace(cfg["static_prefix"]),
		Gateway:   strings.TrimSpace(cfg["static_gateway"]),
		DNS:       strings.TrimSpace(cfg["static_dns"]),
	}
}

func applyNetworkSettings(cfg map[string]string, s networkSettings) {
	cfg["network_mode"] = strings.TrimSpace(strings.ToLower(s.Mode))
	cfg["network_interface"] = strings.TrimSpace(s.Interface)
	cfg["static_address"] = strings.TrimSpace(s.Address)
	cfg["static_prefix"] = strings.TrimSpace(s.Prefix)
	cfg["static_gateway"] = strings.TrimSpace(s.Gateway)
	cfg["static_dns"] = strings.TrimSpace(s.DNS)
}

func validateNetworkSettings(s networkSettings) error {
	mode := strings.TrimSpace(strings.ToLower(s.Mode))
	if mode != "dhcp" && mode != "static" {
		return errors.New("network mode must be 'dhcp' or 'static'")
	}

	iface := strings.TrimSpace(s.Interface)
	if iface != "" && !isSafeInterfaceName(iface) {
		return errors.New("invalid interface")
	}

	if mode == "dhcp" {
		return nil
	}

	if iface == "" {
		return errors.New("static mode requires interface")
	}

	if strings.TrimSpace(s.Address) == "" {
		return errors.New("static mode requires address")
	}
	if strings.TrimSpace(s.Prefix) == "" {
		return errors.New("static mode requires prefix")
	}

	return nil
}

func networkInterfaceDetails(interfaces []string, wireless []string) []networkInterfaceInfo {
	wirelessSet := make(map[string]bool, len(wireless))
	for _, w := range wireless {
		wirelessSet[w] = true
	}

	netIfaces, _ := net.Interfaces()
	ifaceMap := make(map[string]net.Interface, len(netIfaces))
	for _, iface := range netIfaces {
		ifaceMap[iface.Name] = iface
	}

	details := make([]networkInterfaceInfo, 0, len(interfaces))
	for _, name := range interfaces {
		info := networkInterfaceInfo{
			Name:       name,
			IsWireless: wirelessSet[name],
			Addresses:  []string{},
		}

		data, _ := os.ReadFile(filepath.Join("/sys/class/net", name, "operstate"))
		info.Operstate = strings.TrimSpace(string(data))

		if iface, ok := ifaceMap[name]; ok {
			info.MAC = iface.HardwareAddr.String()
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				if ip, _, err := net.ParseCIDR(addr.String()); err == nil && !ip.IsLoopback() {
					info.Addresses = append(info.Addresses, addr.String())
				}
			}
		}

		if info.IsWireless && info.Operstate == "up" {
			info.SSID = wifiSSID(name)
		}

		details = append(details, info)
	}
	return details
}

func listNetworkInterfaces() ([]string, []string) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return []string{}, []string{}
	}

	interfaces := make([]string, 0, len(entries))
	wireless := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		// /sys/class/net entries are symlinks; entry.IsDir() is always false — skip that check
		if name == "" || name == "lo" {
			continue
		}
		interfaces = append(interfaces, name)
		if info, err := os.Stat(filepath.Join("/sys/class/net", name, "wireless")); err == nil && info.IsDir() {
			wireless = append(wireless, name)
		}
	}

	sort.Strings(interfaces)
	sort.Strings(wireless)
	return interfaces, wireless
}

func scanWireGuardUSBConfigs() []wireguardUSBConfig {
	configs := make([]wireguardUSBConfig, 0)
	for _, mount := range usbMountPoints() {
		for _, path := range wireGuardConfigsOnMount(mount) {
			configs = append(configs, newWireGuardUSBConfig(mount, path))
		}
	}
	sort.Slice(configs, func(i, j int) bool {
		if configs[i].Mount == configs[j].Mount {
			return configs[i].Filename < configs[j].Filename
		}
		return configs[i].Mount < configs[j].Mount
	})
	return configs
}

func findWireGuardUSBConfig(path string) (wireguardUSBConfig, bool) {
	for _, config := range scanWireGuardUSBConfigs() {
		if config.Path == path {
			return config, true
		}
	}
	return wireguardUSBConfig{}, false
}

func usbMountPoints() []string {
	cmd := exec.Command("lsblk", "-d", "-o", "NAME,TRAN", "-n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}
	}

	mounts := make([]string, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "usb" {
			continue
		}
		device := "/dev/" + fields[0]
		mountOutput, err := exec.Command("lsblk", "-o", "MOUNTPOINT", "-n", device).CombinedOutput()
		if err != nil {
			continue
		}
		for _, mount := range strings.Split(strings.TrimSpace(string(mountOutput)), "\n") {
			mount = strings.TrimSpace(mount)
			if mount == "" || mount == "/" || mount == "/boot" {
				continue
			}
			mounts = append(mounts, mount)
		}
	}
	return uniqueSortedStrings(mounts)
}

func wireGuardConfigsOnMount(mount string) []string {
	configs := make([]string, 0)
	for _, pattern := range []string{"wg*.conf", "wireguard*.conf"} {
		matches, _ := filepath.Glob(filepath.Join(mount, pattern))
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && info.Mode().IsRegular() {
				configs = append(configs, match)
			}
		}
	}
	return uniqueSortedStrings(configs)
}

func newWireGuardUSBConfig(mount, path string) wireguardUSBConfig {
	filename := filepath.Base(path)
	iface := strings.TrimSuffix(filename, filepath.Ext(filename))
	destination := filepath.Join("/etc/wireguard", filename)
	needsImport := true
	if existing, err := os.ReadFile(destination); err == nil {
		if source, err := os.ReadFile(path); err == nil && bytes.Equal(existing, source) {
			needsImport = false
		}
	}

	return wireguardUSBConfig{
		Path:        path,
		Mount:       mount,
		Filename:    filename,
		Interface:   iface,
		NeedsImport: needsImport,
	}
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		filtered = append(filtered, value)
	}
	sort.Strings(filtered)
	return filtered
}

func applyNetworkFromConfig(configPath string) string {
	if _, err := exec.LookPath("sudo"); err != nil {
		return "network saved; live apply unavailable (sudo not found)"
	}

	cmd := exec.Command("sudo", "-n", "/usr/bin/tc-configure-network", "--apply-tcconfig", configPath, "--reload")
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		log.Printf("network live apply failed: %s", message)
		return "network saved; live apply failed: " + message
	}

	if message == "" {
		return "network saved and applied"
	}

	return "network saved and applied: " + message
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s from %s in %s", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}
