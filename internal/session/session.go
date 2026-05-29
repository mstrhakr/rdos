package session

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var resolutionPattern = regexp.MustCompile(`^([1-9][0-9]{1,4})x([1-9][0-9]{1,4})$`)

type State string

const (
	StateDisconnected State = "disconnected"
	StateConnecting   State = "connecting"
	StateConnected    State = "connected"
	StateError        State = "error"
)

type ConnectRequest struct {
	Server     string   `json:"server"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Domain     string   `json:"domain"`
	CertPolicy string   `json:"certPolicy"`
	Resolution string   `json:"resolution"` // "dynamic" (default fullscreen) or "WxH" (e.g. "1920x1080")
	ExtraArgs  []string `json:"extraArgs"`
}

type Snapshot struct {
	State      State  `json:"state"`
	Message    string `json:"message"`
	ExitCode   int    `json:"exitCode,omitempty"`
	LastOutput string `json:"lastOutput,omitempty"`
	StartedAt  int64  `json:"startedAt,omitempty"`
}

type Manager struct {
	rdpBinary string
	logPath   string

	mu        sync.Mutex
	state     Snapshot
	cancel    context.CancelFunc
	activeCmd *exec.Cmd
	activeLog *os.File
}

func NewManager(rdpBinary, logPath string) *Manager {
	return &Manager{
		rdpBinary: rdpBinary,
		logPath:   logPath,
		state: Snapshot{
			State:   StateDisconnected,
			Message: "idle",
		},
	}
}

func (m *Manager) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Manager) Connect(req ConnectRequest) error {
	if strings.TrimSpace(req.Server) == "" {
		return errors.New("server is required")
	}

	m.mu.Lock()
	if m.activeCmd != nil {
		m.mu.Unlock()
		return errors.New("session already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd, stdin, logFile, err := m.buildCommand(ctx, req)
	if err != nil {
		cancel()
		m.mu.Unlock()
		return err
	}

	m.activeCmd = cmd
	m.activeLog = logFile
	m.cancel = cancel
	m.state = Snapshot{State: StateConnecting, Message: "launching xfreerdp3", StartedAt: time.Now().Unix()}
	m.mu.Unlock()

	if err := cmd.Start(); err != nil {
		cancel()
		m.mu.Lock()
		m.activeCmd = nil
		if m.activeLog != nil {
			_ = m.activeLog.Close()
			m.activeLog = nil
		}
		m.cancel = nil
		m.state = Snapshot{State: StateError, Message: fmt.Sprintf("failed to start session: %v", err)}
		m.mu.Unlock()
		return fmt.Errorf("start xfreerdp3: %w", err)
	}

	m.mu.Lock()
	m.state = Snapshot{State: StateConnected, Message: "session active", StartedAt: time.Now().Unix()}
	m.mu.Unlock()

	if req.Password != "" {
		_, _ = io.WriteString(stdin, req.Password+"\n")
	}
	_ = stdin.Close()

	go m.waitForExit(cmd)
	return nil
}

func (m *Manager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel == nil {
		return nil
	}

	m.cancel()
	m.state = Snapshot{State: StateDisconnected, Message: "session stopped"}
	return nil
}

func (m *Manager) buildCommand(ctx context.Context, req ConnectRequest) (*exec.Cmd, io.WriteCloser, *os.File, error) {
	args := []string{
		"/v:" + strings.TrimSpace(req.Server),
		"+multitouch",
		"/network:auto",
		"+auto-reconnect",
		"+multitransport",
	}

	// Resolution / display mode.
	// "dynamic" (default): fullscreen multimon with dynamic resize — standard thin-client mode.
	// "WxH": windowed at that fixed size with dynamic-resolution so the server still adjusts on resize.
	resolution := strings.TrimSpace(req.Resolution)
	if m := resolutionPattern.FindStringSubmatch(strings.ToLower(resolution)); m != nil {
		args = append(args, "/w:"+m[1], "/h:"+m[2], "/dynamic-resolution")
	} else {
		// Default: fullscreen + multimon + dynamic-resolution.
		args = append(args, "/f", "/multimon", "/dynamic-resolution")
	}

	if req.Username != "" {
		args = append(args, "/u:"+req.Username)
	}
	if req.Domain != "" {
		args = append(args, "/d:"+req.Domain)
	}

	certPolicy := strings.TrimSpace(req.CertPolicy)
	if certPolicy == "" {
		certPolicy = "tofu"
	}
	args = append(args, "/cert:"+certPolicy)

	if req.Password != "" {
		args = append(args, "/from-stdin")
	}
	for _, arg := range req.ExtraArgs {
		if arg == "" {
			continue
		}
		args = append(args, arg)
	}

	cmd := exec.CommandContext(ctx, m.rdpBinary, args...)
	homeDir := strings.TrimSpace(os.Getenv("HOME"))
	if homeDir == "" {
		homeDir = "/home/thinclient"
	}
	xdgConfigHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}
	freerdpDir := filepath.Join(xdgConfigHome, "freerdp")
	if err := os.MkdirAll(freerdpDir, 0o700); err != nil {
		return nil, nil, nil, fmt.Errorf("ensure freerdp config dir: %w", err)
	}
	cmd.Env = append(os.Environ(),
		"HOME="+homeDir,
		"XDG_CONFIG_HOME="+xdgConfigHome,
		"WLOG_APPENDER=file",
		"WLOG_FILEAPPENDER_OUTPUT_FILE_NAME=session.log",
		"WLOG_FILEAPPENDER_OUTPUT_FILE_PATH=.",
	)

	logFile, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open session log: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = logFile.Close()
		return nil, nil, nil, fmt.Errorf("create stdin pipe: %w", err)
	}

	return cmd, stdin, logFile, nil
}

func (m *Manager) waitForExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.activeCmd = nil
	if m.activeLog != nil {
		_ = m.activeLog.Close()
		m.activeLog = nil
	}
	m.cancel = nil

	logOutput := logTail(m.logPath, 8)

	if err == nil {
		m.state = Snapshot{State: StateDisconnected, Message: "session ended", LastOutput: logOutput}
		return
	}

	if errors.Is(err, context.Canceled) {
		m.state = Snapshot{State: StateDisconnected, Message: "session cancelled", LastOutput: logOutput}
		return
	}

	exitCode := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	m.state = Snapshot{State: StateError, Message: err.Error(), ExitCode: exitCode, LastOutput: logOutput}
}

// logTail reads the last n non-empty lines from a log file, stripping WLOG timestamps.
// Returns empty string if the file cannot be read.
func logTail(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return ""
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
