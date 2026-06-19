package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/zwforum/proxy-web/internal/model"
)

// FileStore implements data persistence using JSON/YAML files
type FileStore struct {
	dataDir string
	mu      sync.RWMutex
}

func NewFileStore(dataDir string) *FileStore {
	return &FileStore{dataDir: dataDir}
}

// --- Profile Registry ---

func (s *FileStore) profilesFile() string {
	return filepath.Join(s.dataDir, "webui", "profiles.json")
}

func (s *FileStore) LoadProfileRegistry() (*model.ProfileRegistry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	registry := &model.ProfileRegistry{
		Profiles: []model.Profile{},
	}

	data, err := os.ReadFile(s.profilesFile())
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return nil, fmt.Errorf("read profiles: %w", err)
	}

	if err := json.Unmarshal(data, registry); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	return registry, nil
}

func (s *FileStore) SaveProfileRegistry(registry *model.ProfileRegistry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	return os.WriteFile(s.profilesFile(), data, 0644)
}

// --- Profile-specific files ---

func (s *FileStore) ProfileDir(id string) string {
	return filepath.Join(s.dataDir, "webui", "profiles", id)
}

func (s *FileStore) ReadRules(profileID string) (string, error) {
	path := filepath.Join(s.ProfileDir(profileID), "rules.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *FileStore) WriteRules(profileID string, content string) error {
	dir := s.ProfileDir(profileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "rules.yaml"), []byte(content), 0644)
}

func (s *FileStore) ReadOverride(profileID string) (string, error) {
	path := filepath.Join(s.ProfileDir(profileID), "override.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

func (s *FileStore) WriteOverride(profileID string, content string) error {
	dir := s.ProfileDir(profileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "override.yaml"), []byte(content), 0644)
}

// --- Mihomo config ---

func (s *FileStore) MihomoConfigPath() string {
	return filepath.Join(s.dataDir, "mihomo", "config.yaml")
}

func (s *FileStore) WriteMihomoConfig(content []byte) error {
	return os.WriteFile(s.MihomoConfigPath(), content, 0644)
}

// --- Subscription-level rules/override (new model) ---

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
