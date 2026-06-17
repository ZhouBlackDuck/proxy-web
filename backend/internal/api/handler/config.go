package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/enhance"
	"github.com/zwforum/proxy-web/internal/export"
	"github.com/zwforum/proxy-web/internal/kernel"
	"github.com/zwforum/proxy-web/internal/store"
	"github.com/zwforum/proxy-web/internal/substore"
)

// ConfigHandler handles profile activation, preview, export/import
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
		exporter: export.NewExporter(s, cfg.SubStore.APIAddr),
		kernel:   kernel.NewClient(cfg.Mihomo.APIAddr, cfg.Mihomo.Secret),
		subStore: substore.NewClient(cfg.SubStore.APIAddr),
	}
}

// Activate switches to a profile, merges config, and applies to mihomo
func (h *ConfigHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Build the merged config
	finalYaml, err := h.buildConfig(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "config build failed: " + err.Error(),
		})
		return
	}

	// Validate
	if errors := h.pipeline.Validate(finalYaml); len(errors) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "config validation failed",
			"details": errors,
		})
		return
	}

	// Write to mihomo config file
	if err := h.store.WriteMihomoConfig(finalYaml); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "write config file: " + err.Error(),
		})
		return
	}

	// Apply to mihomo via API
	if err := h.kernel.PutConfig(string(finalYaml)); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "apply to mihomo: " + err.Error(),
		})
		return
	}

	// Update active profile
	registry, err := h.store.LoadProfileRegistry()
	if err == nil {
		registry.ActiveProfileID = id
		h.store.SaveProfileRegistry(registry)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "profile activated"})
}

// Preview returns the merged config without applying it
func (h *ConfigHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	finalYaml, err := h.buildConfig(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Validate
	warnings := h.pipeline.Validate(finalYaml)

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(finalYaml)

	if len(warnings) > 0 {
		// Can't set headers after writing body, log instead
		fmt.Printf("config preview warnings: %v\n", warnings)
	}
}

// Export exports a profile as a zip file
func (h *ConfigHandler) Export(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	zipData, err := h.exporter.Export(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=profile-%s.zip", id))
	w.WriteHeader(http.StatusOK)
	w.Write(zipData)
}

// Import imports a profile from a zip file
func (h *ConfigHandler) Import(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid multipart form",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing file field",
		})
		return
	}
	defer file.Close()

	zipData, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "read file: " + err.Error(),
		})
		return
	}

	// Check if force import subs
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
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// buildConfig runs the full config merge pipeline for a profile
func (h *ConfigHandler) buildConfig(profileID string) ([]byte, error) {
	// Load profile
	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}

	var profileName, subscriptionName string
	for _, p := range registry.Profiles {
		if p.ID == profileID {
			profileName = p.Name
			subscriptionName = p.SubscriptionName
			break
		}
	}
	if profileName == "" {
		return nil, fmt.Errorf("profile %s not found", profileID)
	}

	// 1. Get subscription yaml from Sub-Store
	var subYaml string
	if subscriptionName != "" {
		downloaded, err := h.subStore.DownloadSubscription(subscriptionName, "ClashMeta")
		if err != nil {
			return nil, fmt.Errorf("download subscription %s: %w", subscriptionName, err)
		}
		subYaml = downloaded
	}

	// 2. Read global override
	overrideYaml, _ := h.store.ReadOverride(profileID)

	// 3. Read global rules
	globalRules, _ := h.store.ReadRules(profileID)

	// 4. Build port settings from platform config
	portSettings := map[string]enhance.PortSetting{
		"mixed-port":  {Enabled: h.cfg.Ports.MixedPort.Enabled, Port: h.cfg.Ports.MixedPort.Port},
		"port":        {Enabled: h.cfg.Ports.HTTPPort.Enabled, Port: h.cfg.Ports.HTTPPort.Port},
		"socks-port":  {Enabled: h.cfg.Ports.SocksPort.Enabled, Port: h.cfg.Ports.SocksPort.Port},
		"redir-port":  {Enabled: h.cfg.Ports.RedirPort.Enabled, Port: h.cfg.Ports.RedirPort.Port},
		"tproxy-port": {Enabled: h.cfg.Ports.TProxyPort.Enabled, Port: h.cfg.Ports.TProxyPort.Port},
	}

	// 5. Run pipeline with port settings
	return h.pipeline.BuildWithPorts(subYaml, overrideYaml, globalRules, portSettings)
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

// UpdatePorts updates port settings, saves to settings.json, and re-applies config to kernel
func (h *ConfigHandler) UpdatePorts(w http.ResponseWriter, r *http.Request) {
	var ports config.PortSettings
	if err := json.NewDecoder(r.Body).Decode(&ports); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	// Save to settings
	h.cfg.Ports = ports
	if err := h.cfg.Save(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "save settings: " + err.Error(),
		})
		return
	}

	// If there's an active profile, regenerate and push config
	registry, err := h.store.LoadProfileRegistry()
	if err == nil && registry.ActiveProfileID != "" {
		finalYaml, err := h.buildConfig(registry.ActiveProfileID)
		if err == nil {
			// Write to file and push to kernel
			h.store.WriteMihomoConfig(finalYaml)
			h.kernel.PutConfig(string(finalYaml))
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "ports updated"})
}
