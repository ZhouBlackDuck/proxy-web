package subconverter

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client communicates with the subconverter API
type Client struct {
	baseURL string
	tmpDir  string
	http    *http.Client
}

func NewClient(baseURL string, tmpDir string) *Client {
	// Ensure URL has scheme
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	os.MkdirAll(tmpDir, 0755)
	return &Client{
		baseURL: baseURL,
		tmpDir:  tmpDir,
		http: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // 订阅服务器可能使用自签名证书
				},
			},
		},
	}
}

// FetchResult holds the result of fetching a subscription
type FetchResult struct {
	Content    string // Raw or decoded content
	FilePath   string // Local file path for subconverter (if content was saved)
	IsClash    bool   // Whether content is Clash format
	IsBase64   bool   // Whether content was base64 encoded
}

// FetchRaw fetches the raw content of a subscription URL or local file
// For base64 content, decodes and saves to temp file for subconverter.
// The name parameter is used in the temp filename for per-subscription cleanup.
// The ua parameter overrides the default User-Agent for HTTP requests.
func (c *Client) FetchRaw(input string, name string, ua string) (*FetchResult, error) {
	var rawContent string

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		req, err := http.NewRequest("GET", input, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch raw: %w", err)
		}
		// Use custom UA if set, otherwise fall back to a common subscription client UA
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		} else {
			req.Header.Set("User-Agent", "clash-verge/v2.4.7")
		}
		req.Header.Set("Accept", "*/*")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch raw: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("fetch raw: %d %s", resp.StatusCode, string(body))
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch raw: read: %w", err)
		}
		rawContent = string(data)
	} else {
		// Local file path: read directly
		data, err := os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("read local file: %w", err)
		}
		rawContent = string(data)
	}

	result := &FetchResult{
		Content:  rawContent,
		IsClash:  IsClashFormat(rawContent),
	}

	// If not Clash format, try base64 decode
	if !result.IsClash {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rawContent))
		if err == nil && len(decoded) > 0 {
			result.IsBase64 = true
			result.Content = string(decoded)
			// Check if decoded content is Clash format
			result.IsClash = IsClashFormat(result.Content)
		}
	}

	// For non-Clash content, save to temp file for subconverter
	if !result.IsClash {
		tmpFile := filepath.Join(c.tmpDir, "sub_"+name+"_"+time.Now().Format("20060102150405")+".txt")
		os.WriteFile(tmpFile, []byte(result.Content), 0644)
		result.FilePath = tmpFile
	}

	return result, nil
}

// IsClashFormat checks if content is standard Clash YAML (with proxies section)
func IsClashFormat(content string) bool {
	return strings.Contains(content, "proxies:") || strings.Contains(content, "Proxy:")
}

// Convert converts a subscription URL to Clash (mihomo) format via subconverter
// For remote URLs, let subconverter download directly (avoids temp file issues)
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
