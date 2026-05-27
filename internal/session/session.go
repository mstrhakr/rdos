package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

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
	ExtraArgs  []string `json:"extraArgs"`
}

type Snapshot struct {
	State     State  `json:"state"`
	Message   string `json:"message"`
	StartedAt int64  `json:"startedAt,omitempty"`
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
		"/multimon",
		"+multitouch",
		"/f",
		"/network:auto",
		"/dynamic-resolution",
		"+auto-reconnect",
		"+multitransport",
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
	cmd.Env = append(os.Environ(),
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

	if err == nil {
		m.state = Snapshot{State: StateDisconnected, Message: "session ended"}
		return
	}

	if errors.Is(err, context.Canceled) {
		m.state = Snapshot{State: StateDisconnected, Message: "session cancelled"}
		return
	}

	m.state = Snapshot{State: StateError, Message: err.Error()}
}
