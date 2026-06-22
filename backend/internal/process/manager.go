package process

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/zwforum/proxy-web/internal/config"
)

// Manager manages child processes (mihomo and subconverter)
type Manager struct {
	cfg         *config.Config
	mihomo      *processInfo
	subconverter *processInfo
	mu          sync.RWMutex
}

type processInfo struct {
	cmd      *exec.Cmd
	running  bool
	pid      int
	started  time.Time
	restarts int
	stopCh   chan struct{} // signals that stop was intentional
	mu       sync.Mutex
}

// ProcessStatus is the public status of a managed process
type ProcessStatus struct {
	Name     string `json:"name"`
	Running  bool   `json:"running"`
	PID      int    `json:"pid"`
	Uptime   int64  `json:"uptime"` // seconds
	Restarts int    `json:"restarts"`
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:         cfg,
		mihomo:      &processInfo{stopCh: make(chan struct{}, 1)},
		subconverter: &processInfo{stopCh: make(chan struct{}, 1)},
	}
}

// StartMihomo starts the mihomo process
func (m *Manager) StartMihomo() error {
	m.mihomo.mu.Lock()
	defer m.mihomo.mu.Unlock()

	if m.mihomo.running {
		return fmt.Errorf("mihomo already running")
	}

	binaryPath := m.cfg.Mihomo.BinaryPath
	configPath := m.cfg.Mihomo.ConfigPath

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("mihomo binary not found: %s", binaryPath)
	}

	args := []string{"-d", filepath.Dir(configPath)}

	cmd := exec.Command(binaryPath, args...)

	// Redirect stdout/stderr to log file (append mode)
	logPath := filepath.Join(m.cfg.DataDir, "mihomo", "mihomo.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("start mihomo: %w", err)
	}

	m.mihomo.cmd = cmd
	m.mihomo.running = true
	m.mihomo.pid = cmd.Process.Pid
	m.mihomo.started = time.Now()
	m.mihomo.stopCh = make(chan struct{}, 1)

	// Wait for health check
	if err := waitForHTTP(fmt.Sprintf("http://%s/", m.cfg.Mihomo.APIAddr), 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "mihomo health check failed: %v\n", err)
	}

	// Monitor process in background
	go m.monitorProcess("mihomo", m.mihomo, logFile)

	fmt.Printf("mihomo started (PID: %d)\n", cmd.Process.Pid)
	return nil
}

// StopMihomo stops the mihomo process
func (m *Manager) StopMihomo() error {
	m.mihomo.mu.Lock()
	defer m.mihomo.mu.Unlock()

	if !m.mihomo.running || m.mihomo.cmd == nil {
		return nil
	}

	select {
	case m.mihomo.stopCh <- struct{}{}:
	default:
	}
	m.mihomo.running = false

	if m.mihomo.cmd.Process != nil {
		m.mihomo.cmd.Process.Signal(syscall.SIGTERM)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			if m.mihomo.cmd.Process != nil {
				m.mihomo.cmd.Process.Kill()
			}
			fmt.Println("mihomo force killed")
			m.mihomo.pid = 0
			return nil
		default:
			if m.mihomo.pid == 0 {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// StartSubConverter starts the subconverter process
func (m *Manager) StartSubConverter() error {
	m.subconverter.mu.Lock()
	defer m.subconverter.mu.Unlock()

	if m.subconverter.running {
		return fmt.Errorf("subconverter already running")
	}

	binaryPath := m.cfg.SubConverter.BinaryPath
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("subconverter binary not found: %s", binaryPath)
	}

	// Redirect subconverter output to log file with size limit
	logPath := filepath.Join(m.cfg.DataDir, "subconverter", "subconverter.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)

	// Rotate log if exceeds 10MB
	const maxLogSize = 10 * 1024 * 1024
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxLogSize {
		os.Remove(logPath)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open subconverter log: %w", err)
	}

	cmd := exec.Command(binaryPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start subconverter: %w", err)
	}

	m.subconverter.cmd = cmd
	m.subconverter.running = true
	m.subconverter.pid = cmd.Process.Pid
	m.subconverter.started = time.Now()
	m.subconverter.stopCh = make(chan struct{}, 1)

	// Wait for health check
	if err := waitForHTTP(fmt.Sprintf("http://%s/version", m.cfg.SubConverter.APIAddr), 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "subconverter health check failed: %v\n", err)
	}

	go m.monitorProcess("subconverter", m.subconverter, nil)

	fmt.Printf("subconverter started (PID: %d)\n", cmd.Process.Pid)
	return nil
}

// StopSubConverter stops the subconverter process
func (m *Manager) StopSubConverter() error {
	m.subconverter.mu.Lock()
	defer m.subconverter.mu.Unlock()

	if !m.subconverter.running || m.subconverter.cmd == nil {
		return nil
	}

	select {
	case m.subconverter.stopCh <- struct{}{}:
	default:
	}
	m.subconverter.running = false

	if m.subconverter.cmd.Process != nil {
		m.subconverter.cmd.Process.Signal(syscall.SIGTERM)
	}

	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			if m.subconverter.cmd.Process != nil {
				m.subconverter.cmd.Process.Kill()
			}
			fmt.Println("subconverter force killed")
			m.subconverter.pid = 0
			return nil
		default:
			if m.subconverter.pid == 0 {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// StopAll stops all managed processes
func (m *Manager) StopAll() {
	m.StopMihomo()
	m.StopSubConverter()
}

// Status returns the status of all managed processes
func (m *Manager) Status() []ProcessStatus {
	statuses := make([]ProcessStatus, 0, 2)

	m.mihomo.mu.Lock()
	uptime := int64(0)
	if m.mihomo.running {
		uptime = int64(time.Since(m.mihomo.started).Seconds())
	}
	statuses = append(statuses, ProcessStatus{
		Name:     "mihomo",
		Running:  m.mihomo.running,
		PID:      m.mihomo.pid,
		Uptime:   uptime,
		Restarts: m.mihomo.restarts,
	})
	m.mihomo.mu.Unlock()

	m.subconverter.mu.Lock()
	uptime = int64(0)
	if m.subconverter.running {
		uptime = int64(time.Since(m.subconverter.started).Seconds())
	}
	statuses = append(statuses, ProcessStatus{
		Name:     "subconverter",
		Running:  m.subconverter.running,
		PID:      m.subconverter.pid,
		Uptime:   uptime,
		Restarts: m.subconverter.restarts,
	})
	m.subconverter.mu.Unlock()

	return statuses
}

// MihomoAlive returns whether mihomo is running
func (m *Manager) MihomoAlive() bool {
	m.mihomo.mu.Lock()
	defer m.mihomo.mu.Unlock()
	return m.mihomo.running
}

// SubConverterAlive returns whether subconverter is running
func (m *Manager) SubConverterAlive() bool {
	m.subconverter.mu.Lock()
	defer m.subconverter.mu.Unlock()
	return m.subconverter.running
}

// monitorProcess watches a process and restarts it on unexpected exit
func (m *Manager) monitorProcess(name string, pi *processInfo, logFile *os.File) {
	if pi.cmd == nil {
		return
	}

	pi.cmd.Wait()

	if logFile != nil {
		logFile.Close()
	}

	pi.mu.Lock()
	wasRunning := pi.running
	pi.running = false
	pi.pid = 0
	pi.mu.Unlock()

	select {
	case <-pi.stopCh:
		fmt.Printf("%s stopped\n", name)
		return
	default:
	}

	if wasRunning {
		fmt.Printf("%s exited unexpectedly\n", name)

		if pi.restarts < 3 {
			pi.restarts++
			backoff := time.Duration(pi.restarts) * 5 * time.Second
			fmt.Printf("restarting %s in %v (attempt %d/3)\n", name, backoff, pi.restarts)
			time.Sleep(backoff)

			switch name {
			case "mihomo":
				if restartErr := m.StartMihomo(); restartErr != nil {
					fmt.Fprintf(os.Stderr, "failed to restart %s: %v\n", name, restartErr)
				}
			case "subconverter":
				if restartErr := m.StartSubConverter(); restartErr != nil {
					fmt.Fprintf(os.Stderr, "failed to restart %s: %v\n", name, restartErr)
				}
			}
		} else {
			fmt.Printf("%s exceeded max restart attempts\n", name)
		}
	}
}

// waitForHTTP polls an HTTP endpoint until it responds or timeout
func waitForHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for %s", url)
}
