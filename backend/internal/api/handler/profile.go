package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/zwforum/proxy-web/internal/model"
	"github.com/zwforum/proxy-web/internal/store"
)

type ProfileHandler struct {
	store *store.FileStore
}

func NewProfileHandler(store *store.FileStore) *ProfileHandler {
	return &ProfileHandler{store: store}
}

// List returns all profiles
func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, registry)
}

// Create creates a new profile
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "name is required",
		})
		return
	}

	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	profile := model.Profile{
		ID:               fmt.Sprintf("p-%d", time.Now().UnixNano()),
		Name:             req.Name,
		Description:      req.Description,
		SubscriptionName: req.SubscriptionName,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		ExportSettings:   model.ExportSettings{IncludeSubscriptions: false},
	}

	registry.Profiles = append(registry.Profiles, profile)

	// Auto-activate if first profile
	if len(registry.Profiles) == 1 {
		registry.ActiveProfileID = profile.ID
	}

	if err := h.store.SaveProfileRegistry(registry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Create profile directory with empty rules and override
	h.store.WriteRules(profile.ID, "")
	h.store.WriteOverride(profile.ID, "")

	writeJSON(w, http.StatusCreated, profile)
}

// Get returns a profile by ID
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	for _, p := range registry.Profiles {
		if p.ID == id {
			writeJSON(w, http.StatusOK, p)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "profile not found",
	})
}

// Update updates a profile
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req model.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	found := false
	for i := range registry.Profiles {
		if registry.Profiles[i].ID == id {
			found = true
			if req.Name != nil {
				registry.Profiles[i].Name = *req.Name
			}
			if req.Description != nil {
				registry.Profiles[i].Description = *req.Description
			}
			if req.SubscriptionName != nil {
				registry.Profiles[i].SubscriptionName = *req.SubscriptionName
			}
			if req.ExportSettings != nil {
				registry.Profiles[i].ExportSettings = *req.ExportSettings
			}
			registry.Profiles[i].UpdatedAt = time.Now()
			break
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "profile not found",
		})
		return
	}

	if err := h.store.SaveProfileRegistry(registry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// Delete removes a profile
func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	newProfiles := make([]model.Profile, 0, len(registry.Profiles))
	for _, p := range registry.Profiles {
		if p.ID != id {
			newProfiles = append(newProfiles, p)
		}
	}

	if len(newProfiles) == len(registry.Profiles) {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "profile not found",
		})
		return
	}

	registry.Profiles = newProfiles
	if registry.ActiveProfileID == id {
		if len(newProfiles) > 0 {
			registry.ActiveProfileID = newProfiles[0].ID
		} else {
			registry.ActiveProfileID = ""
		}
	}

	if err := h.store.SaveProfileRegistry(registry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// Activate switches to a profile
func (h *ProfileHandler) Activate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	registry, err := h.store.LoadProfileRegistry()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	found := false
	for _, p := range registry.Profiles {
		if p.ID == id {
			found = true
			break
		}
	}

	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "profile not found",
		})
		return
	}

	registry.ActiveProfileID = id
	if err := h.store.SaveProfileRegistry(registry); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// TODO: Trigger config merge and apply to mihomo
	writeJSON(w, http.StatusOK, map[string]string{"message": "activated"})
}

// GetRules returns the global rules for a profile
func (h *ProfileHandler) GetRules(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rules, err := h.store.ReadRules(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(rules))
}

// UpdateRules updates the global rules for a profile
func (h *ProfileHandler) UpdateRules(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.store.WriteRules(id, body.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "rules updated"})
}

// GetOverride returns the global override for a profile
func (h *ProfileHandler) GetOverride(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	override, err := h.store.ReadOverride(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(override))
}

// UpdateOverride updates the global override for a profile
func (h *ProfileHandler) UpdateOverride(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := h.store.WriteOverride(id, body.Content); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "override updated"})
}
