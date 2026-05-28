package main

import (
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

type networkInterfacesResponse struct {
	Interfaces       []string `json:"interfaces"`
	Wireless         []string `json:"wireless"`
	HasWireless      bool     `json:"hasWireless"`
	DefaultInterface string   `json:"defaultInterface"`
	DefaultWireless  string   `json:"defaultWireless"`
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
