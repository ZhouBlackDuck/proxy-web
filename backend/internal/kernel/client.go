package kernel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a mihomo REST API client
type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func NewClient(baseURL, secret string) *Client {
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// --- Types ---

type Version struct {
	Premium bool   `json:"premium"`
	Meta    bool   `json:"meta"`
	Version string `json:"version"`
}

type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

type Memory struct {
	Inuse   uint64 `json:"inuse"`
	OSLimit uint64 `json:"oslimit"`
}

type Config struct {
	Port               int    `json:"port" yaml:"port"`
	SocksPort          int    `json:"socks-port" yaml:"socks-port"`
	RedirPort          int    `json:"redir-port" yaml:"redir-port"`
	TProxyPort         int    `json:"tproxy-port" yaml:"tproxy-port"`
	MixedPort          int    `json:"mixed-port" yaml:"mixed-port"`
	AllowLan           bool   `json:"allow-lan" yaml:"allow-lan"`
	BindAddress        string `json:"bind-address" yaml:"bind-address"`
	Mode               string `json:"mode" yaml:"mode"`
	LogLevel           string `json:"log-level" yaml:"log-level"`
	IPv6               bool   `json:"ipv6" yaml:"ipv6"`
	Sniffing           bool   `json:"sniffing" yaml:"sniffing"`
	TcpConcurrent      bool   `json:"tcp-concurrent" yaml:"tcp-concurrent"`
	InterfaceName      string `json:"interface-name" yaml:"interface-name"`
	Tun                TunConfig `json:"tun" yaml:"tun"`
}

type TunConfig struct {
	Enable     bool   `json:"enable" yaml:"enable"`
	Stack      string `json:"stack" yaml:"stack"`
	AutoRoute  bool   `json:"auto-route" yaml:"auto-route"`
}

type Proxy struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	History []DelayHistory `json:"history"`
	Now     string   `json:"now,omitempty"`
	All     []string `json:"all,omitempty"`
}

type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay int       `json:"delay"`
}

type ProxiesResponse struct {
	Proxies map[string]Proxy `json:"proxies"`
}

type Rule struct {
	Index   int        `json:"index"`
	Type    string     `json:"type"`
	Payload string     `json:"payload"`
	Proxy   string     `json:"proxy"`
	Size    int        `json:"size"`
	Extra   *RuleExtra `json:"extra,omitempty"`
}

type RuleExtra struct {
	Disabled  bool      `json:"disabled"`
	HitCount  uint64    `json:"hitCount"`
}

type RulesResponse struct {
	Rules []Rule `json:"rules"`
}

type ConnectionMeta struct {
	Network     string `json:"network"`
	Type        string `json:"type"`
	SourceIP    string `json:"sourceIP"`
	DestIP      string `json:"destinationIP"`
	SourcePort  string `json:"sourcePort"`
	DestPort    string `json:"destinationPort"`
	Host        string `json:"host"`
	DNSMode     string `json:"dnsMode"`
	ProcessPath string `json:"processPath"`
}

type Connection struct {
	ID         string         `json:"id"`
	Metadata   ConnectionMeta `json:"metadata"`
	Upload     int64          `json:"upload"`
	Download   int64          `json:"download"`
	Start      time.Time      `json:"start"`
	Chains     []string       `json:"chains"`
	Rule       string         `json:"rule"`
	RulePayload string        `json:"rulePayload"`
}

type ConnectionsSnapshot struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []Connection `json:"connections"`
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

	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
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
		return fmt.Errorf("mihomo API %s %s: %d %s", method, path, resp.StatusCode, string(data))
	}

	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// GetVersion returns mihomo version info
func (c *Client) GetVersion() (*Version, error) {
	var v Version
	if err := c.doJSON("GET", "/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetConfigs returns the current running config
func (c *Client) GetConfigs() (*Config, error) {
	var cfg Config
	if err := c.doJSON("GET", "/configs", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// PatchConfig incrementally updates config
func (c *Client) PatchConfig(patch map[string]interface{}) error {
	return c.doJSON("PATCH", "/configs", patch, nil)
}

// PutConfig reloads config from a file path on disk
func (c *Client) PutConfig(configPath string) error {
	body := map[string]string{"path": configPath}
	return c.doJSON("PUT", "/configs", body, nil)
}

// UpdateGeo updates GeoIP/GeoSite databases
func (c *Client) UpdateGeo() error {
	return c.doJSON("POST", "/configs/geo", nil, nil)
}

// GetProxies returns all proxies
func (c *Client) GetProxies() (*ProxiesResponse, error) {
	var resp ProxiesResponse
	if err := c.doJSON("GET", "/proxies", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProxy returns a specific proxy
func (c *Client) GetProxy(name string) (*Proxy, error) {
	var p Proxy
	if err := c.doJSON("GET", "/proxies/"+name, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SwitchProxy switches the selected node in a Selector group
func (c *Client) SwitchProxy(groupName, nodeName string) error {
	return c.doJSON("PUT", "/proxies/"+groupName, map[string]string{"name": nodeName}, nil)
}

// GetGroups returns all proxy groups
func (c *Client) GetGroups() (*ProxiesResponse, error) {
	// mihomo /group returns proxies as an array, not a map
	var raw struct {
		Proxies []Proxy `json:"proxies"`
	}
	if err := c.doJSON("GET", "/group", nil, &raw); err != nil {
		return nil, err
	}
	// Convert array to map for consistent frontend handling
	proxies := make(map[string]Proxy)
	for _, p := range raw.Proxies {
		proxies[p.Name] = p
	}
	return &ProxiesResponse{Proxies: proxies}, nil
}

// GetRules returns all rules
func (c *Client) GetRules() (*RulesResponse, error) {
	var resp RulesResponse
	if err := c.doJSON("GET", "/rules", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetConnections returns active connections
func (c *Client) GetConnections() (*ConnectionsSnapshot, error) {
	var snap ConnectionsSnapshot
	if err := c.doJSON("GET", "/connections", nil, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// CloseAllConnections closes all connections
func (c *Client) CloseAllConnections() error {
	return c.doJSON("DELETE", "/connections", nil, nil)
}

// CloseConnection closes a specific connection
func (c *Client) CloseConnection(id string) error {
	return c.doJSON("DELETE", "/connections/"+id, nil, nil)
}

// Restart restarts the mihomo kernel
func (c *Client) Restart() error {
	return c.doJSON("POST", "/restart", nil, nil)
}
