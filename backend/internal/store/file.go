package store

import (
	"os"
	"path/filepath"
	"sync"
)

// FileStore implements data persistence using JSON/YAML files
type FileStore struct {
	dataDir string
	mu      sync.RWMutex
}

func NewFileStore(dataDir string) *FileStore {
	return &FileStore{dataDir: dataDir}
}

// --- Mihomo config ---

func (s *FileStore) MihomoConfigPath() string {
	return filepath.Join(s.dataDir, "mihomo", "config.yaml")
}

func (s *FileStore) WriteMihomoConfig(content []byte) error {
	return os.WriteFile(s.MihomoConfigPath(), content, 0644)
}

// --- Subscription-level rules/override ---

func (s *FileStore) SubDir(subName string) string {
	return filepath.Join(s.dataDir, "webui", "subscriptions", subName)
}

func (s *FileStore) ReadSubRules(subName string) (string, error) {
	path := filepath.Join(s.SubDir(subName), "rules.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *FileStore) WriteSubRules(subName string, content string) error {
	dir := s.SubDir(subName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "rules.yaml"), []byte(content), 0644)
}

func (s *FileStore) ReadSubOverride(subName string) (string, error) {
	path := filepath.Join(s.SubDir(subName), "override.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *FileStore) WriteSubOverride(subName string, content string) error {
	dir := s.SubDir(subName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "override.yaml"), []byte(content), 0644)
}
