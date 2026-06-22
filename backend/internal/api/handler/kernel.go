package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/kernel"
)

// KernelHandler proxies requests to the mihomo kernel API
type KernelHandler struct {
	cfg    *config.Config
	client *kernel.Client
}

func NewKernelHandler(cfg *config.Config) *KernelHandler {
	return &KernelHandler{
		cfg:    cfg,
		client: kernel.NewClient(cfg.Mihomo.APIAddr, cfg.Mihomo.Secret),
	}
}

// Proxy forwards requests to mihomo API
func (h *KernelHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	// Strip /api/kernel prefix and forward to mihomo
	mihomoPath := strings.TrimPrefix(r.URL.Path, "/api/kernel")
	if mihomoPath == "" {
		mihomoPath = "/"
	}

	// Build the mihomo request URL
	targetURL := fmt.Sprintf("http://%s%s", h.cfg.Mihomo.APIAddr, mihomoPath)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create proxy request: " + err.Error(),
		})
		return
	}

	// Copy headers
	for k, v := range r.Header {
		proxyReq.Header[k] = v
	}
	if h.cfg.Mihomo.Secret != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+h.cfg.Mihomo.Secret)
	}
	proxyReq.Header.Set("Host", h.cfg.Mihomo.APIAddr)

	// Execute
	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "mihomo unreachable: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// GetVersion returns mihomo version
func (h *KernelHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	v, err := h.client.GetVersion()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// GetConfigs returns current config
func (h *KernelHandler) GetConfigs(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.client.GetConfigs()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// PatchConfig updates config incrementally
func (h *KernelHandler) PatchConfig(w http.ResponseWriter, r *http.Request) {
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.client.PatchConfig(patch); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// PutConfig reloads the entire config
func (h *KernelHandler) PutConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Payload string `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.client.PutConfig(body.Payload); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// GetProxies returns all proxies
func (h *KernelHandler) GetProxies(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.client.GetProxies()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, proxies)
}

// GetGroups returns all proxy groups
func (h *KernelHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.client.GetGroups()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

// GetRules returns all rules
func (h *KernelHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.client.GetRules()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// GetConnections returns active connections
func (h *KernelHandler) GetConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := h.client.GetConnections()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, conns)
}

// CloseAllConnections closes all connections
func (h *KernelHandler) CloseAllConnections(w http.ResponseWriter, r *http.Request) {
	if err := h.client.CloseAllConnections(); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// Restart restarts the mihomo kernel
func (h *KernelHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.client.Restart(); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ok"})
}

// --- GeoIP Handler ---

// GeoStatus returns GeoIP/GeoSite database status
func (h *KernelHandler) GeoStatus(w http.ResponseWriter, r *http.Request) {
	type geoFile struct {
		Name      string `json:"name"`
		Exists    bool   `json:"exists"`
		Size      int64  `json:"size"`
		UpdatedAt string `json:"updatedAt"`
	}

	files := []string{"geoip.metadb", "geosite.dat", "geoip.dat"}
	var status []geoFile

	for _, name := range files {
		path := filepath.Join(h.cfg.DataDir, "mihomo", name)
		info, err := os.Stat(path)
		if err != nil {
			status = append(status, geoFile{Name: name, Exists: false})
		} else {
			status = append(status, geoFile{
				Name:      name,
				Exists:    true,
				Size:      info.Size(),
				UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"files": status,
	})
}

// GeoUpdate triggers GeoIP/GeoSite database update
func (h *KernelHandler) GeoUpdate(w http.ResponseWriter, r *http.Request) {
	if err := h.client.UpdateGeo(); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "GeoIP/GeoSite update triggered"})
}

