package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"github.com/zwforum/proxy-web/internal/subconverter"
	"github.com/zwforum/proxy-web/internal/subscription"
)

// SubscriptionHandler handles subscription CRUD and conversion
type SubscriptionHandler struct {
	store       *subscription.Store
	converter   *subconverter.Client
	tmpDir      string
}

func NewSubscriptionHandler(store *subscription.Store, converter *subconverter.Client, dataDir string) *SubscriptionHandler {
	tmpDir := filepath.Join(dataDir, "webui", "tmp")
	os.MkdirAll(tmpDir, 0755)
	return &SubscriptionHandler{
		store:     store,
		converter: converter,
		tmpDir:    tmpDir,
	}
}

// List returns all subscriptions
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.store.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": subs})
}

// Create creates a new subscription
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sub subscription.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if sub.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := h.store.Create(sub); err != nil {
		code := http.StatusInternalServerError
		if _, ok := err.(*subscription.DuplicateError); ok {
			code = http.StatusConflict
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "created"})
}

// Get returns a subscription by name
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	sub, err := h.store.Get(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscription not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": sub})
}

// Update updates a subscription
func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.store.Update(name, patch); err != nil {
		code := http.StatusInternalServerError
		if _, ok := err.(*subscription.NotFoundError); ok {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// Delete deletes a subscription and its temp files
func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.store.Delete(name); err != nil {
		code := http.StatusInternalServerError
		if _, ok := err.(*subscription.NotFoundError); ok {
			code = http.StatusNotFound
		}
		writeJSON(w, code, map[string]string{"error": err.Error()})
		return
	}
	// Clean up temp files for this subscription
	h.cleanupTempFiles(name)
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// cleanupTempFiles removes temporary files for a subscription
func (h *SubscriptionHandler) cleanupTempFiles(name string) {
	pattern := filepath.Join(h.tmpDir, "sub_*.txt")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, f := range files {
		os.Remove(f)
	}
}

// Sync validates a subscription by converting it through subconverter
func (h *SubscriptionHandler) Sync(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	sub, err := h.store.Get(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscription not found"})
		return
	}

	input := h.resolveInput(sub)
	if _, err := h.converter.Convert(input); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "sync ok"})
}

// Download returns the converted subscription config
func (h *SubscriptionHandler) Download(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	sub, err := h.store.Get(name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscription not found"})
		return
	}

	input := h.resolveInput(sub)
	config, err := h.converter.Convert(input)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(config))
}

// resolveInput returns the subconverter input: URL for remote, temp file path for local
func (h *SubscriptionHandler) resolveInput(sub *subscription.Subscription) string {
	if sub.Source == "url" && sub.URL != "" {
		return sub.URL
	}
	// Local subscription: write content to a temp file and return the path
	tmpFile := filepath.Join(h.tmpDir, sub.Name+".yaml")
	os.WriteFile(tmpFile, []byte(sub.Content), 0644)
	return tmpFile
}
