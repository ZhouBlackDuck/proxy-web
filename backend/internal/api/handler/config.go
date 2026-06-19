package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-chi/chi/v5"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/enhance"
	"github.com/zwforum/proxy-web/internal/export"
	"github.com/zwforum/proxy-web/internal/kernel"
	"github.com/zwforum/proxy-web/internal/store"
	"github.com/zwforum/proxy-web/internal/substore"
)

// ConfigHandler handles subscription activation, preview, rules/override, export/import
type ConfigHandler struct {
	cfg      *config.Config
	store    *store.FileStore
	pipeline *enhance.Pipeline
	exporter *export.Exporter
	kernel   *kernel.Client
	subStore *substore.Client
}

func NewConfigHandler(cfg *config.Config, s *store.FileStore) *ConfigHandler {
	return &ConfigHandler{
		cfg:      cfg,
		store:    s,
		pipeline: enhance.NewPipeline(),
		exporter: export.NewExporter(s, cfg.SubStore.APIAddr, cfg),
		kernel:   kernel.NewClient(cfg.Mihomo.APIAddr, cfg.Mihomo.Secret),
		subStore: substore.NewClient(cfg.SubStore.APIAddr),
	}
}

// Activate builds merged config for a subscription and pushes it to mihomo
func (h *ConfigHandler) Activate(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")

	finalYaml, err := h.buildConfig(subName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "config build failed: " + err.Error(),
		})
		return
	}

	if errors := h.pipeline.Validate(finalYaml); len(errors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "config validation failed",
			"details": errors,
		})
		return
	}

	if err := h.store.WriteMihomoConfig(finalYaml); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "write config file: " + err.Error(),
		})
		return
	}

	if err := h.kernel.PutConfig(string(finalYaml)); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "apply to mihomo: " + err.Error(),
		})
		return
	}

	// PATCH runtime settings that PUT /configs may not apply
	h.kernel.PatchConfig(map[string]interface{}{
		"allow-lan":    true,
		"bind-address": "*",
	})

	// Save active subscription
	h.cfg.ActiveSubscription = subName
	h.cfg.Save()

	writeJSON(w, http.StatusOK, map[string]string{"message": "subscription activated"})
}

// Preview returns the merged config without applying it
func (h *ConfigHandler) Preview(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")

	finalYaml, err := h.buildConfig(subName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(finalYaml)
}

// GetSubRules returns global rules for a subscription
func (h *ConfigHandler) GetSubRules(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")
	content, _ := h.store.ReadSubRules(subName)
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// UpdateSubRules saves global rules for a subscription
func (h *ConfigHandler) UpdateSubRules(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.store.WriteSubRules(subName, body.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "rules saved"})
}

// GetSubOverride returns global override for a subscription
func (h *ConfigHandler) GetSubOverride(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")
	content, _ := h.store.ReadSubOverride(subName)
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

// UpdateSubOverride saves global override for a subscription
func (h *ConfigHandler) UpdateSubOverride(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.store.WriteSubOverride(subName, body.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "override saved"})
}

// Export exports a subscription as a zip file
func (h *ConfigHandler) Export(w http.ResponseWriter, r *http.Request) {
	subName := chi.URLParam(r, "name")

	zipData, err := h.exporter.ExportSubscription(subName)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=sub-%s.zip", subName))
	w.WriteHeader(http.StatusOK)
	w.Write(zipData)
}

// ExportAll exports all platform config + optionally all subscriptions
func (h *ConfigHandler) ExportAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TestSites []map[string]interface{} `json:"testSites"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	zipData, err := h.exporter.ExportAll(req.TestSites)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=proxy-web-config-%s.zip", time.Now().Format("2006-01-02")))
	w.WriteHeader(http.StatusOK)
	w.Write(zipData)
}

// Import imports subscriptions from a zip file
func (h *ConfigHandler) Import(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	zipData, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read file: " + err.Error()})
		return
	}

	var forceImportSubs *bool
	if r.FormValue("importSubscriptions") == "true" {
		t := true
		forceImportSubs = &t
	} else if r.FormValue("importSubscriptions") == "false" {
		f := false
		forceImportSubs = &f
	}

	result, err := h.exporter.Import(zipData, forceImportSubs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetActiveSubscription returns the currently active subscription name
func (h *ConfigHandler) GetActiveSubscription(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"activeSubscription": h.cfg.ActiveSubscription,
	})
}

// buildConfig runs the full config merge pipeline for a subscription
// Pipeline order: subscription → rules(prepend) → override(merge) → defaults → ports
func (h *ConfigHandler) buildConfig(subscriptionName string) ([]byte, error) {
	// 1. Get subscription yaml from Sub-Store
	var subYaml string
	if subscriptionName != "" && subscriptionName != "__empty__" {
		downloaded, err := h.subStore.GetRawContent(subscriptionName)
		if err != nil {
			// If subscription not found, use minimal empty config
			subYaml = "proxies: []\nproxy-groups: []\nrules: []"
		} else {
			subYaml = downloaded
		}
	} else {
		// Empty subscription - use minimal empty config
		subYaml = "proxies: []\nproxy-groups: []\nrules: []"
	}

	// 2. Read global rules (stored under __global__ key)
	globalRules, _ := h.store.ReadSubRules("__global__")

	// 3. Read global override (stored under __global__ key)
	overrideYaml, _ := h.store.ReadSubOverride("__global__")

	// 4. Build port settings from platform config
	portSettings := map[string]enhance.PortSetting{
		"mixed-port":  {Enabled: h.cfg.Ports.MixedPort.Enabled, Port: h.cfg.Ports.MixedPort.Port},
		"port":        {Enabled: h.cfg.Ports.HTTPPort.Enabled, Port: h.cfg.Ports.HTTPPort.Port},
		"socks-port":  {Enabled: h.cfg.Ports.SocksPort.Enabled, Port: h.cfg.Ports.SocksPort.Port},
		"redir-port":  {Enabled: h.cfg.Ports.RedirPort.Enabled, Port: h.cfg.Ports.RedirPort.Port},
		"tproxy-port": {Enabled: h.cfg.Ports.TProxyPort.Enabled, Port: h.cfg.Ports.TProxyPort.Port},
	}

	// 5. Run pipeline: subscription → rules → override → defaults → ports
	finalYaml, err := h.pipeline.BuildWithPorts(subYaml, overrideYaml, globalRules, portSettings)
	if err != nil {
		return nil, err
	}

	// 6. Override external-controller and secret to ensure platform can always connect
	var configMap map[string]interface{}
	if err := yaml.Unmarshal(finalYaml, &configMap); err == nil {
		configMap["external-controller"] = h.cfg.Mihomo.APIAddr
		if h.cfg.Mihomo.Secret != "" {
			configMap["secret"] = h.cfg.Mihomo.Secret
		}
		if overridden, err := yaml.Marshal(configMap); err == nil {
			finalYaml = overridden
		}
	}

	return finalYaml, nil
}

// ValidateConfig validates a yaml config
func (h *ConfigHandler) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	errors := h.pipeline.Validate([]byte(body.Content))
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid":  len(errors) == 0,
		"errors": errors,
	})
}

// GetPorts returns current port settings
func (h *ConfigHandler) GetPorts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.cfg.Ports)
}

// UpdatePorts updates port settings and re-applies config if a subscription is active
func (h *ConfigHandler) UpdatePorts(w http.ResponseWriter, r *http.Request) {
	var ports config.PortSettings
	if err := json.NewDecoder(r.Body).Decode(&ports); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	h.cfg.Ports = ports
	if err := h.cfg.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "save settings: " + err.Error(),
		})
		return
	}

	// If there's an active subscription, regenerate and push config
	if h.cfg.ActiveSubscription != "" {
		finalYaml, err := h.buildConfig(h.cfg.ActiveSubscription)
		if err == nil {
			h.store.WriteMihomoConfig(finalYaml)
			h.kernel.PutConfig(string(finalYaml))
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "ports updated"})
}

// GetExportSetting returns the platform-level export include subscriptions setting
func (h *ConfigHandler) GetExportSetting(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"includeSubscriptions": h.cfg.ExportIncludeSubscriptions,
	})
}

// UpdateExportSetting updates the platform-level export setting
func (h *ConfigHandler) UpdateExportSetting(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IncludeSubscriptions bool `json:"includeSubscriptions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	h.cfg.ExportIncludeSubscriptions = body.IncludeSubscriptions
	if err := h.cfg.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "export setting updated"})
}
