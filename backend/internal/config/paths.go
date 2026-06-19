package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultDataDir = "/data"
	DefaultAPIPort = 3000
)

// Config represents the application configuration loaded from settings.json
type Config struct {
	DataDir                    string        `json:"-"`
	PasswordHash               string        `json:"passwordHash,omitempty"`
	Theme                      string        `json:"theme"`
	Language                   string        `json:"language"`
	Mihomo                     MihomoConfig  `json:"mihomo"`
	SubStore                   SubStoreConfig `json:"substore"`
	Ports                      PortSettings  `json:"ports"`
	ActiveSubscription         string        `json:"activeSubscription,omitempty"`
	ExportIncludeSubscriptions bool          `json:"exportIncludeSubscriptions"`
}

type MihomoConfig struct {
	APIAddr    string `json:"apiAddr"`
	Secret     string `json:"secret"`
	BinaryPath string `json:"binaryPath"`
	ConfigPath string `json:"configPath"`
}

type SubStoreConfig struct {
	APIAddr string `json:"apiAddr"`
	DataDir string `json:"dataDir"`
}

// PortSettings controls which ports are enabled and their values
type PortSettings struct {
	MixedPort  PortEntry `json:"mixedPort"`
	HTTPPort   PortEntry `json:"httpPort"`
	SocksPort  PortEntry `json:"socksPort"`
	RedirPort  PortEntry `json:"redirPort"`
	TProxyPort PortEntry `json:"tproxyPort"`
}

type PortEntry struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig(dataDir string) *Config {
	return &Config{
		DataDir:  dataDir,
		Theme:    "dark",
		Language: "zh",
		Ports: PortSettings{
			MixedPort:  PortEntry{Enabled: true, Port: 7890},
			HTTPPort:   PortEntry{Enabled: false, Port: 7891},
			SocksPort:  PortEntry{Enabled: false, Port: 7892},
			RedirPort:  PortEntry{Enabled: false, Port: 7893},
			TProxyPort: PortEntry{Enabled: false, Port: 7894},
		},
		Mihomo: MihomoConfig{
			APIAddr:    "127.0.0.1:9090",
			Secret:     "",
			BinaryPath: filepath.Join(dataDir, "mihomo", "bin", "mihomo"),
			ConfigPath: filepath.Join(dataDir, "mihomo", "config.yaml"),
		},
		SubStore: SubStoreConfig{
			APIAddr: "127.0.0.1:3001",
			DataDir: filepath.Join(dataDir, "sub-store"),
		},
	}
}

// Load reads settings.json from the data directory
func Load() (*Config, error) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = DefaultDataDir
	}

	settingsPath := filepath.Join(dataDir, "webui", "settings.json")

	cfg := DefaultConfig(dataDir)

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Use defaults, will be saved on first setup
			return cfg, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	cfg.DataDir = dataDir

	return cfg, nil
}

// Save writes the config to settings.json
func (c *Config) Save() error {
	settingsPath := filepath.Join(c.DataDir, "webui", "settings.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	return nil
}

// InitDirs creates all necessary data directories
func InitDirs(cfg *Config) error {
	dirs := []string{
		filepath.Join(cfg.DataDir, "webui", "profiles"),
		filepath.Join(cfg.DataDir, "mihomo", "bin"),
		cfg.SubStore.DataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	return nil
}

// Paths returns commonly used paths
type Paths struct {
	DataDir       string
	WebUIDir      string
	ProfilesDir   string
	MihomoDir     string
	SubStoreDir   string
	SettingsFile  string
	ProfilesFile  string
}

func GetPaths(dataDir string) *Paths {
	return &Paths{
		DataDir:      dataDir,
		WebUIDir:     filepath.Join(dataDir, "webui"),
		ProfilesDir:  filepath.Join(dataDir, "webui", "profiles"),
		MihomoDir:    filepath.Join(dataDir, "mihomo"),
		SubStoreDir:  filepath.Join(dataDir, "sub-store"),
		SettingsFile: filepath.Join(dataDir, "webui", "settings.json"),
		ProfilesFile: filepath.Join(dataDir, "webui", "profiles.json"),
	}
}
