package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/substore"
)

// SubscriptionHandler proxies subscription requests to Sub-Store
type SubscriptionHandler struct {
	cfg    *config.Config
	client *substore.Client
}

func NewSubscriptionHandler(cfg *config.Config) *SubscriptionHandler {
	return &SubscriptionHandler{
		cfg:    cfg,
		client: substore.NewClient(cfg.SubStore.APIAddr),
	}
}

// List returns all subscriptions
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.client.ListSubscriptions()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": subs})
}

// Create creates a new subscription
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var sub substore.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if sub.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := h.client.CreateSubscription(sub); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"message": "created"})
}

// Get returns a subscription by name
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/subscriptions/")
	sub, err := h.client.GetSubscription(name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"data": sub})
}

// Update updates a subscription
func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/subscriptions/")
	var patch map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if err := h.client.UpdateSubscription(name, patch); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// Delete deletes a subscription
func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := extractPathParam(r.URL.Path, "/api/subscriptions/")
	if err := h.client.DeleteSubscription(name); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// Sync triggers a subscription sync
func (h *SubscriptionHandler) Sync(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
	name = strings.TrimSuffix(name, "/sync")
	if err := h.client.SyncSubscription(name); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "sync triggered"})
}

// Download gets the converted subscription config
func (h *SubscriptionHandler) Download(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
	name = strings.TrimSuffix(name, "/download")
	target := r.URL.Query().Get("target")
	if target == "" {
		target = "ClashMeta"
	}

	config, err := h.client.DownloadSubscription(name, target)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(config))
}

// Flow returns subscription flow/usage info
func (h *SubscriptionHandler) Flow(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
	name = strings.TrimSuffix(name, "/flow")
	info, err := h.client.GetFlowInfo(name)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, info)
}
