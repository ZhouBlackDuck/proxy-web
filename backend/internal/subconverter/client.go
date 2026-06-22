package subconverter

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client communicates with the subconverter API
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	// Ensure URL has scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// FetchRaw fetches the raw content of a subscription URL or local file without conversion
func (c *Client) FetchRaw(input string) (string, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		resp, err := c.http.Get(input)
		if err != nil {
			return "", fmt.Errorf("fetch raw: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return "", fmt.Errorf("fetch raw: %d %s", resp.StatusCode, string(body))
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("fetch raw: read: %w", err)
		}
		return string(data), nil
	}
	// Local file path: read directly
	data, err := os.ReadFile(input)
	if err != nil {
		return "", fmt.Errorf("read local file: %w", err)
	}
	return string(data), nil
}

// IsClashFormat checks if content looks like Clash YAML
func IsClashFormat(content string) bool {
	return strings.Contains(content, "proxies:") || strings.Contains(content, "Proxy:")
}

// Convert converts a subscription (URL or local file path) to Clash (mihomo) format via subconverter
func (c *Client) Convert(input string) (string, error) {
	reqURL := fmt.Sprintf("%s/sub?target=clash&url=%s", c.baseURL, url.QueryEscape(input))
	resp, err := c.http.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("subconverter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("subconverter: %d %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("subconverter: read: %w", err)
	}
	return string(data), nil
}

// IsAlive checks if subconverter is reachable
func (c *Client) IsAlive() bool {
	resp, err := c.http.Get(c.baseURL + "/version")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}
