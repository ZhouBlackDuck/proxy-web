package substore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client communicates with the Sub-Store backend API
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Types ---

type Subscription struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName,omitempty"`
	URL         string            `json:"url,omitempty"`
	Source      string            `json:"source,omitempty"` // "url" or "local"
	Content     string            `json:"content,omitempty"`
	UA          string            `json:"ua,omitempty"`
	Process     []ProcessItem     `json:"process,omitempty"`
	UpdatedAt   int64             `json:"updatedAt,omitempty"`
}

type ProcessItem struct {
	Type  string                 `json:"type"`
	Args  map[string]interface{} `json:"args,omitempty"`
}

type Collection struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName,omitempty"`
	Subscriptions []string `json:"subscriptions"`
	Process     []ProcessItem `json:"process,omitempty"`
}

// --- API Methods ---

func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := fmt.Sprintf("http://%s%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.http.Do(req)
}

func (c *Client) doJSON(method, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sub-store API %s %s: %d %s", method, path, resp.StatusCode, string(data))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// ListSubscriptions returns all subscriptions
func (c *Client) ListSubscriptions() ([]Subscription, error) {
	var resp struct {
		Data []Subscription `json:"data"`
	}
	if err := c.doJSON("GET", "/api/subs", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetSubscription returns a specific subscription
func (c *Client) GetSubscription(name string) (*Subscription, error) {
	var resp struct {
		Data Subscription `json:"data"`
	}
	if err := c.doJSON("GET", "/api/sub/"+name, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// GetRawContent returns the raw content of a local subscription
// For local subscriptions, this returns the stored content directly
// without going through format conversion
func (c *Client) GetRawContent(name string) (string, error) {
	sub, err := c.GetSubscription(name)
	if err != nil {
		return "", err
	}
	if sub.Source == "local" && sub.Content != "" {
		return sub.Content, nil
	}
	// For URL subscriptions, fall back to ClashMeta conversion
	return c.DownloadSubscription(name, "ClashMeta")
}

// CreateSubscription creates a new subscription
func (c *Client) CreateSubscription(sub Subscription) error {
	return c.doJSON("POST", "/api/subs", sub, nil)
}

// UpdateSubscription updates a subscription
func (c *Client) UpdateSubscription(name string, patch map[string]interface{}) error {
	return c.doJSON("PATCH", "/api/sub/"+name, patch, nil)
}

// DeleteSubscription deletes a subscription
func (c *Client) DeleteSubscription(name string) error {
	return c.doJSON("DELETE", "/api/sub/"+name, nil, nil)
}

// SyncSubscription triggers a sync for a subscription
func (c *Client) SyncSubscription(name string) error {
	return c.doJSON("POST", "/api/sync/"+name, nil, nil)
}

// DownloadSubscription gets the converted config in the specified target format
func (c *Client) DownloadSubscription(name, target string) (string, error) {
	url := fmt.Sprintf("http://%s/download/%s?target=%s", c.baseURL, name, target)
	resp, err := c.http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed: %d %s", resp.StatusCode, string(data))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	return string(data), nil
}

// GetFlowInfo returns subscription flow/usage info
func (c *Client) GetFlowInfo(name string) (map[string]interface{}, error) {
	var resp map[string]interface{}
	if err := c.doJSON("GET", "/api/sub/flow/"+name, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// --- Collections ---

// ListCollections returns all collections
func (c *Client) ListCollections() ([]Collection, error) {
	var resp struct {
		Data []Collection `json:"data"`
	}
	if err := c.doJSON("GET", "/api/collections", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CreateCollection creates a new collection (multi-sub merge)
func (c *Client) CreateCollection(col Collection) error {
	return c.doJSON("POST", "/api/collections", col, nil)
}

// DeleteCollection deletes a collection
func (c *Client) DeleteCollection(name string) error {
	return c.doJSON("DELETE", "/api/collection/"+name, nil, nil)
}

// DownloadCollection gets the converted collection config
func (c *Client) DownloadCollection(name, target string) (string, error) {
	url := fmt.Sprintf("http://%s/download/collection/%s?target=%s", c.baseURL, name, target)
	resp, err := c.http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed: %d %s", resp.StatusCode, string(data))
	}

	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

// IsAlive checks if Sub-Store is reachable
func (c *Client) IsAlive() bool {
	resp, err := c.doRequest("GET", "/api/subs", nil)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
