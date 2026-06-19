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

// Manager manages child processes (mihomo and Sub-Store)
type Manager struct {
	cfg      *config.Config
	mihomo   *processInfo
	substore *processInfo
	mu       sync.RWMutex
}

type processInfo struct {
	cmd     *exec.Cmd
	running bool
	pid     int
	started time.Time
	restarts int
	stopCh  chan struct{} // signals that stop was intentional
	mu      sync.Mutex
}

// ProcessStatus is the public status of a managed process
type ProcessStatus struct {
	Name    string  `json:"name"`
	Running bool    `json:"running"`
	PID     int     `json:"pid"`
	Uptime  int64   `json:"uptime"` // seconds
	Restarts int    `json:"restarts"`
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg:      cfg,
		mihomo:   &processInfo{stopCh: make(chan struct{}, 1)},
		substore: &processInfo{stopCh: make(chan struct{}, 1)},
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

	// Mark as intentional stop (prevents auto-restart)
	select {
	case m.mihomo.stopCh <- struct{}{}:
	default:
	}
	m.mihomo.running = false

	// Send SIGTERM
	if m.mihomo.cmd.Process != nil {
		m.mihomo.cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait for process to exit (monitored by goroutine)
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
				return nil // monitor already cleaned up
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// StartSubStore starts the Sub-Store Node.js service
func (m *Manager) StartSubStore() error {
	m.substore.mu.Lock()
	defer m.substore.mu.Unlock()

	if m.substore.running {
		return fmt.Errorf("sub-store already running")
	}

	// Sub-Store entry point
	subStoreEntry := "/app/sub-store/sub-store.bundle.js"
	if _, err := os.Stat(subStoreEntry); os.IsNotExist(err) {
		subStoreEntry = "/app/sub-store/sub-store.min.js"
		if _, err := os.Stat(subStoreEntry); os.IsNotExist(err) {
			subStoreEntry = "/app/sub-store/src/main.js"
			if _, err := os.Stat(subStoreEntry); os.IsNotExist(err) {
				return fmt.Errorf("sub-store entry not found")
			}
		}
	}

	cmd := exec.Command("node", subStoreEntry)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SUB_STORE_BACKEND_API_PORT=%s", portFromAddr(m.cfg.SubStore.APIAddr)),
		fmt.Sprintf("SUB_STORE_BACKEND_API_HOST=%s", hostFromAddr(m.cfg.SubStore.APIAddr)),
		fmt.Sprintf("SUB_STORE_DATA_DIR=%s", m.cfg.SubStore.DataDir),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start sub-store: %w", err)
	}

	m.substore.cmd = cmd
	m.substore.running = true
	m.substore.pid = cmd.Process.Pid
	m.substore.started = time.Now()
	m.substore.stopCh = make(chan struct{}, 1)

	// Wait for health check
	if err := waitForHTTP(fmt.Sprintf("http://%s/api/subs", m.cfg.SubStore.APIAddr), 30*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "sub-store health check failed: %v\n", err)
	}

	go m.monitorProcess("sub-store", m.substore, nil)

	fmt.Printf("sub-store started (PID: %d)\n", cmd.Process.Pid)
	return nil
}

// StopSubStore stops the Sub-Store process
func (m *Manager) StopSubStore() error {
	m.substore.mu.Lock()
	defer m.substore.mu.Unlock()

	if !m.substore.running || m.substore.cmd == nil {
		return nil
	}

	// Mark as intentional stop
	select {
	case m.substore.stopCh <- struct{}{}:
	default:
	}
	m.substore.running = false

	if m.substore.cmd.Process != nil {
		m.substore.cmd.Process.Signal(syscall.SIGTERM)
	}

	// Wait for process to exit
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			if m.substore.cmd.Process != nil {
				m.substore.cmd.Process.Kill()
			}
			fmt.Println("sub-store force killed")
			m.substore.pid = 0
			return nil
		default:
			if m.substore.pid == 0 {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

// StopAll stops all managed processes
func (m *Manager) StopAll() {
	m.StopMihomo()
	m.StopSubStore()
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

	m.substore.mu.Lock()
	uptime = int64(0)
	if m.substore.running {
		uptime = int64(time.Since(m.substore.started).Seconds())
	}
	statuses = append(statuses, ProcessStatus{
		Name:     "sub-store",
		Running:  m.substore.running,
		PID:      m.substore.pid,
		Uptime:   uptime,
		Restarts: m.substore.restarts,
	})
	m.substore.mu.Unlock()

	return statuses
}

// MihomoAlive returns whether mihomo is running
func (m *Manager) MihomoAlive() bool {
	m.mihomo.mu.Lock()
	defer m.mihomo.mu.Unlock()
	return m.mihomo.running
}

// SubStoreAlive returns whether Sub-Store is running
func (m *Manager) SubStoreAlive() bool {
	m.substore.mu.Lock()
	defer m.substore.mu.Unlock()
	return m.substore.running
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

	// Check if this was an intentional stop
	select {
	case <-pi.stopCh:
		// Intentional stop, don't restart
		fmt.Printf("%s stopped\n", name)
		return
	default:
	}

	if wasRunning {
		fmt.Printf("%s exited unexpectedly\n", name)

		// Auto-restart with exponential backoff (max 3 times)
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
			case "sub-store":
				if restartErr := m.StartSubStore(); restartErr != nil {
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

// portFromAddr extracts port from "host:port"
func portFromAddr(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return addr
}

// hostFromAddr extracts host from "host:port"
func hostFromAddr(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
