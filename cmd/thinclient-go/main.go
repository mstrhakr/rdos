package main

import (
	"bytes"
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
)

type networkInterfacesResponse struct {
	Interfaces       []string `json:"interfaces"`
	Wireless         []string `json:"wireless"`
	HasWireless      bool     `json:"hasWireless"`
	DefaultInterface string   `json:"defaultInterface"`
	DefaultWireless  string   `json:"defaultWireless"`
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

type wireguardStatusResponse struct {
	Enabled    bool     `json:"enabled"`
	HasConfig  bool     `json:"hasConfig"`
	Interfaces []string `json:"interfaces"`
}

type wireguardEnableRequest struct {
	Enabled bool `json:"enabled"`
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

	mu     sync.Mutex
	config map[string]string
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

	mux := http.NewServeMux()
	mux.Handle("/api/v1/health", application.loopbackOnly(http.HandlerFunc(application.handleHealth)))
	mux.Handle("/api/v1/config", application.loopbackOnly(http.HandlerFunc(application.handleConfig)))
	mux.Handle("/api/v1/network", application.loopbackOnly(http.HandlerFunc(application.handleNetwork)))
	mux.Handle("/api/v1/network/interfaces", application.loopbackOnly(http.HandlerFunc(application.handleNetworkInterfaces)))
	mux.Handle("/api/v1/ota", application.loopbackOnly(http.HandlerFunc(application.handleOTAStatus)))
	mux.Handle("/api/v1/ota/releases", application.loopbackOnly(http.HandlerFunc(application.handleOTAReleases)))
	mux.Handle("/api/v1/ota/update", application.loopbackOnly(http.HandlerFunc(application.handleOTAUpdate)))
	mux.Handle("/api/v1/ota/rollback", application.loopbackOnly(http.HandlerFunc(application.handleOTARollback)))
	mux.Handle("/api/v1/wireguard/usb", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardUSBScan)))
	mux.Handle("/api/v1/wireguard/import", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardUSBImport)))
	mux.Handle("/api/v1/wireguard/enable", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardEnable)))
	mux.Handle("/api/v1/wireguard", application.loopbackOnly(http.HandlerFunc(application.handleWireGuardStatus)))
	mux.Handle("/api/v1/wifi/disconnect", application.loopbackOnly(http.HandlerFunc(application.handleWifiDisconnect)))
	mux.Handle("/api/v1/status", application.loopbackOnly(http.HandlerFunc(application.handleStatus)))
	mux.Handle("/api/v1/wifi/scan", application.loopbackOnly(http.HandlerFunc(application.handleWifiScan)))
	mux.Handle("/api/v1/wifi/connect", application.loopbackOnly(http.HandlerFunc(application.handleWifiConnect)))
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
	respondJSON(w, http.StatusOK, a.sessions.Snapshot())
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

func (a *app) handleWireGuardStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.Lock()
	cfg := cloneConfig(a.config)
	a.mu.Unlock()

	enabled := strings.ToLower(strings.TrimSpace(cfg["wireguard_enabled"])) == "true"

	hasConfigOut, _ := exec.Command("sudo", "-n", "/usr/bin/tc-configure-wireguard", "--has-config").Output()
	hasConfig := strings.TrimSpace(string(hasConfigOut)) == "true"

	var interfaces []string
	if out, err := exec.Command("wg", "show", "interfaces").Output(); err == nil {
		for _, iface := range strings.Fields(string(out)) {
			if iface != "" {
				interfaces = append(interfaces, iface)
			}
		}
	}
	if interfaces == nil {
		interfaces = []string{}
	}

	respondJSON(w, http.StatusOK, wireguardStatusResponse{
		Enabled:    enabled,
		HasConfig:  hasConfig,
		Interfaces: interfaces,
	})
}

func (a *app) handleWireGuardEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req wireguardEnableRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	stateStr := "false"
	if req.Enabled {
		stateStr = "true"
	}

	cmd := exec.Command("sudo", "-n", "/usr/bin/tc-configure-wireguard", "--set-enabled", stateStr)
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		http.Error(w, "wireguard toggle failed: "+message, http.StatusBadRequest)
		return
	}

	actualState := "false"
	if strings.EqualFold(strings.TrimSpace(message), "true") {
		actualState = "true"
	}
	a.mu.Lock()
	updated := cloneConfig(a.config)
	updated["wireguard_enabled"] = actualState
	a.mu.Unlock()

	if err := a.store.Save(updated); err != nil {
		http.Error(w, "failed to persist config", http.StatusInternalServerError)
		return
	}

	a.mu.Lock()
	a.config = updated
	a.mu.Unlock()

	var interfaces []string
	if out, err2 := exec.Command("wg", "show", "interfaces").Output(); err2 == nil {
		for _, iface := range strings.Fields(string(out)) {
			if iface != "" {
				interfaces = append(interfaces, iface)
			}
		}
	}
	if interfaces == nil {
		interfaces = []string{}
	}

	respondJSON(w, http.StatusOK, wireguardStatusResponse{
		Enabled:    actualState == "true",
		HasConfig:  actualState == "true" || strings.EqualFold(strings.TrimSpace(message), "true"),
		Interfaces: interfaces,
	})
}

func (a *app) handleWifiDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmd := exec.Command("sudo", "-n", "/usr/bin/tc-configure-wifi", "--disconnect")
	output, err := cmd.CombinedOutput()
	message := strings.TrimSpace(string(output))
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		http.Error(w, "wifi disconnect failed: "+message, http.StatusBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": message})
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

	if err := a.sessions.Connect(req); err != nil {
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

	for _, candidate := range []string{filepath.Join("/boot/slots", slot+"-version")} {
		if b, err := os.ReadFile(candidate); err == nil {
			return strings.TrimSpace(string(b))
		}
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
		return release.Prerelease || isRC
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

func listNetworkInterfaces() ([]string, []string) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return []string{}, []string{}
	}

	interfaces := make([]string, 0, len(entries))
	wireless := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
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
