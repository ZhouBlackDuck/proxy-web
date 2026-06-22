package handler

import (
	"crypto/sha256"
	"fmt"

	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zwforum/proxy-web/internal/config"
)

// IconHandler handles icon upload and management
type IconHandler struct {
	cfg *config.Config
}

func NewIconHandler(cfg *config.Config) *IconHandler {
	return &IconHandler{cfg: cfg}
}

// UploadIcon uploads an SVG icon and stores it on the server
func (h *IconHandler) UploadIcon(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".svg") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only SVG files are allowed"})
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
		return
	}

	// Create icons directory if not exists
	iconsDir := filepath.Join(h.cfg.DataDir, "webui", "icons")
	if err := os.MkdirAll(iconsDir, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create icons directory"})
		return
	}

	// Generate filename from content hash
	hash := sha256.Sum256(content)
	filename := fmt.Sprintf("icon_%x.svg", hash[:6])
	iconPath := filepath.Join(iconsDir, filename)

	// Skip write if identical file already exists
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		if err := os.WriteFile(iconPath, content, 0644); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save icon"})
			return
		}
	}

	// Return icon URL
	iconURL := fmt.Sprintf("/api/icons/%s", filename)
	writeJSON(w, http.StatusOK, map[string]string{
		"url":      iconURL,
		"filename": filename,
	})
}

// GetIcon serves an icon file
func (h *IconHandler) GetIcon(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		filename = r.URL.Query().Get("file")
	}

	if filename == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing filename"})
		return
	}

	// Security: prevent path traversal
	filename = filepath.Base(filename)
	filepath := filepath.Join(h.cfg.DataDir, "webui", "icons", filename)

	content, err := os.ReadFile(filepath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "icon not found"})
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(content)
}

// ListIcons lists all uploaded icons
func (h *IconHandler) ListIcons(w http.ResponseWriter, r *http.Request) {
	iconsDir := filepath.Join(h.cfg.DataDir, "webui", "icons")
	files := getFiles(iconsDir)

	icons := make([]map[string]string, 0, len(files))
	for _, f := range files {
		icons = append(icons, map[string]string{
			"filename": f,
			"url":      fmt.Sprintf("/api/icons/%s", f),
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"icons": icons,
	})
}

// DeleteIcon deletes an icon
func (h *IconHandler) DeleteIcon(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("file")
	if filename == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing filename"})
		return
	}

	filename = filepath.Base(filename)
	filepath := filepath.Join(h.cfg.DataDir, "webui", "icons", filename)

	if err := os.Remove(filepath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "icon not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "icon deleted"})
}

func getFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".svg") {
			files = append(files, e.Name())
		}
	}
	return files
}
