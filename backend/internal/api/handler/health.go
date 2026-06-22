package handler

import (
	"net/http"
	"time"

	"github.com/zwforum/proxy-web/internal/process"
)

type HealthHandler struct {
	pm *process.Manager
}

func NewHealthHandler(pm *process.Manager) *HealthHandler {
	return &HealthHandler{pm: pm}
}

// Health returns the WebUI backend health status
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":       "ok",
		"mihomo":       h.pm.MihomoAlive(),
		"subconverter": h.pm.SubConverterAlive(),
	})
}

// Status returns detailed process status
func (h *HealthHandler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"processes": h.pm.Status(),
	})
}

// StartMihomo starts the mihomo process
func (h *HealthHandler) StartMihomo(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StartMihomo(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mihomo started"})
}

// StopMihomo stops the mihomo process
func (h *HealthHandler) StopMihomo(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StopMihomo(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mihomo stopped"})
}

// RestartMihomo restarts the mihomo process
func (h *HealthHandler) RestartMihomo(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StopMihomo(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "stop failed: " + err.Error(),
		})
		return
	}
	time.Sleep(1 * time.Second)
	if err := h.pm.StartMihomo(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "start failed: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "mihomo restarted"})
}

// StartSubConverter starts the subconverter process
func (h *HealthHandler) StartSubConverter(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StartSubConverter(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "subconverter started"})
}

// StopSubConverter stops the subconverter process
func (h *HealthHandler) StopSubConverter(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StopSubConverter(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "subconverter stopped"})
}

// RestartSubConverter restarts the subconverter process
func (h *HealthHandler) RestartSubConverter(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.StopSubConverter(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "stop failed: " + err.Error(),
		})
		return
	}
	time.Sleep(500 * time.Millisecond)
	if err := h.pm.StartSubConverter(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "start failed: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "subconverter restarted"})
}
