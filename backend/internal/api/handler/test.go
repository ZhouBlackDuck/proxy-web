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
	{Name: "YouTube", URL: "https://www.youtube.com", Icon: "📺"},
	{Name: "GitHub", URL: "https://api.github.com", Icon: "🐙"},
	{Name: "Cloudflare", URL: "https://1.1.1.1/cdn-cgi/trace", Icon: "☁️"},
	{Name: "Baidu", URL: "https://www.baidu.com", Icon: "🅱️"},
	{Name: "Bilibili", URL: "https://www.bilibili.com", Icon: "📺"},
}

// TestAll tests connectivity to all default sites through the proxy
func (h *TestHandler) TestAll(w http.ResponseWriter, r *http.Request) {
	// Parse custom sites from query or use defaults
	sites := DefaultSites

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

	// Create HTTP client that uses the mihomo proxy
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(fmt.Sprintf("http://%s", h.mihomoAddr))),
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
