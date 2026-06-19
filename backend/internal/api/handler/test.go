package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// TestHandler handles website connectivity testing
type TestHandler struct {
	mihomoAddr string
	secret     string
}

func NewTestHandler(mihomoAddr, secret string) *TestHandler {
	return &TestHandler{
		mihomoAddr: mihomoAddr,
		secret:     secret,
	}
}

// getProxyAddr queries mihomo for the mixed-port to use as proxy
func (h *TestHandler) getProxyAddr() string {
	client := &http.Client{Timeout: 3 * time.Second}
	url := fmt.Sprintf("http://%s/configs", h.mihomoAddr)
	req, _ := http.NewRequest("GET", url, nil)
	if h.secret != "" {
		req.Header.Set("Authorization", "Bearer "+h.secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "127.0.0.1:7890" // fallback
	}
	defer resp.Body.Close()

	var cfg struct {
		MixedPort int `json:"mixed-port"`
		Port      int `json:"port"`
		SocksPort int `json:"socks-port"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return "127.0.0.1:7890"
	}

	port := cfg.MixedPort
	if port == 0 {
		port = cfg.Port
	}
	if port == 0 {
		port = cfg.SocksPort
	}
	if port == 0 {
		port = 7890
	}
	return fmt.Sprintf("127.0.0.1:%d", port)
}

type TestSite struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

type TestResult struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Icon    string `json:"icon"`
	OK      bool   `json:"ok"`
	Latency int    `json:"latency"` // milliseconds, -1 if failed
	Error   string `json:"error,omitempty"`
}

// DefaultSites returns the default list of test sites
var DefaultSites = []TestSite{
	{Name: "Google", URL: "https://www.google.com/generate_204", Icon: "🔍"},
	{Name: "GitHub", URL: "https://api.github.com", Icon: "🐙"},
}

// TestAll tests connectivity to sites through the proxy
// Accepts optional JSON body with custom sites array
func (h *TestHandler) TestAll(w http.ResponseWriter, r *http.Request) {
	sites := DefaultSites

	// Try to parse custom sites from request body
	if r.Body != nil && r.ContentLength > 0 {
		var req struct {
			Sites []TestSite `json:"sites"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Sites) > 0 {
			sites = req.Sites
		}
	}

	results := make([]TestResult, len(sites))
	var wg sync.WaitGroup

	for i, site := range sites {
		wg.Add(1)
		go func(idx int, s TestSite) {
			defer wg.Done()
			results[idx] = h.testSite(s)
		}(i, site)
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
	})
}

// TestSingle tests connectivity to a single URL
func (h *TestHandler) TestSingle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url is required"})
		return
	}

	result := h.testSite(TestSite{Name: req.URL, URL: req.URL, Icon: "🌐"})
	writeJSON(w, http.StatusOK, result)
}

func (h *TestHandler) testSite(site TestSite) TestResult {
	result := TestResult{
		Name:    site.Name,
		URL:     site.URL,
		Icon:    site.Icon,
		Latency: -1,
	}

	proxyAddr := h.getProxyAddr()

	// Create HTTP client that uses the mihomo proxy
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(fmt.Sprintf("http://%s", proxyAddr))),
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	start := time.Now()
	resp, err := client.Get(site.URL)
	elapsed := time.Since(start)

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	latencyMs := int(elapsed.Milliseconds())
	result.Latency = latencyMs

	// 2xx and 3xx are considered reachable
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.OK = true
	} else {
		result.OK = true // Still reachable even with 4xx/5xx
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
