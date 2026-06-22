package subscription

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Subscription represents a proxy subscription
type Subscription struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	URL         string `json:"url,omitempty"`
	Source      string `json:"source"` // "url" or "local"
	Content     string `json:"content,omitempty"`
	UA          string `json:"ua,omitempty"`
	UpdatedAt   int64  `json:"updatedAt,omitempty"`
}

// Store manages subscription persistence using a JSON file
type Store struct {
	dataDir string
	path    string
	mu      sync.RWMutex
}

func NewStore(dataDir string) *Store {
	dir := filepath.Join(dataDir, "webui")
	os.MkdirAll(dir, 0755)
	return &Store{
		dataDir: dataDir,
		path:    filepath.Join(dir, "subscriptions.json"),
	}
}

func (s *Store) List() ([]Subscription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Subscription{}, nil
		}
		return nil, err
	}

	var subs []Subscription
	if err := json.Unmarshal(data, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *Store) Get(name string) (*Subscription, error) {
	subs, err := s.List()
	if err != nil {
		return nil, err
	}
	for i := range subs {
		if subs[i].Name == name {
			return &subs[i], nil
		}
	}
	return nil, nil
}

func (s *Store) Save(subs []Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Create(sub Subscription) error {
	subs, err := s.List()
	if err != nil {
		return err
	}
	for _, existing := range subs {
		if existing.Name == sub.Name {
			return &DuplicateError{Name: sub.Name}
		}
	}
	sub.UpdatedAt = time.Now().Unix()
	subs = append(subs, sub)
	return s.Save(subs)
}

func (s *Store) Update(name string, patch map[string]interface{}) error {
	subs, err := s.List()
	if err != nil {
		return err
	}
	for i := range subs {
		if subs[i].Name == name {
			if v, ok := patch["displayName"]; ok {
				subs[i].DisplayName, _ = v.(string)
			}
			if v, ok := patch["url"]; ok {
				subs[i].URL, _ = v.(string)
			}
			if v, ok := patch["source"]; ok {
				subs[i].Source, _ = v.(string)
			}
			if v, ok := patch["content"]; ok {
				subs[i].Content, _ = v.(string)
			}
			if v, ok := patch["ua"]; ok {
				subs[i].UA, _ = v.(string)
			}
			subs[i].UpdatedAt = time.Now().Unix()
			return s.Save(subs)
		}
	}
	return &NotFoundError{Name: name}
}

func (s *Store) Delete(name string) error {
	subs, err := s.List()
	if err != nil {
		return err
	}
	filtered := make([]Subscription, 0, len(subs))
	found := false
	for _, sub := range subs {
		if sub.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, sub)
	}
	if !found {
		return &NotFoundError{Name: name}
	}
	return s.Save(filtered)
}

type DuplicateError struct{ Name string }

func (e *DuplicateError) Error() string { return "subscription " + e.Name + " already exists" }

type NotFoundError struct{ Name string }

func (e *NotFoundError) Error() string { return "subscription " + e.Name + " not found" }
