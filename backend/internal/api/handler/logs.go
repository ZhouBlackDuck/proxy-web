package handler

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/zwforum/proxy-web/internal/process"
)

// LogHandler serves mihomo logs from the log file
type LogHandler struct {
	pm *process.Manager
}

func NewLogHandler(pm *process.Manager) *LogHandler {
	return &LogHandler{pm: pm}
}

// LogEntry is a structured log entry
type LogEntry struct {
	Time    string `json:"time"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// logrus logfmt pattern: time="..." level=... msg="..."
var logPattern = regexp.MustCompile(`^time="([^"]*)" level=(\w+) msg="(.*)"$`)

// GetLogs returns the last N lines from the mihomo log file, parsed into structured entries
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 200
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	level := r.URL.Query().Get("level")

	lines, err := h.pm.ReadLogs(limit * 2) // read extra to account for filtering
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		entry := parseLogLine(line)
		if entry == nil {
			continue
		}
		// Level filter
		if level != "" && level != "all" && !strings.EqualFold(entry.Type, level) {
			continue
		}
		entries = append(entries, *entry)
		if len(entries) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  entries,
		"total": len(entries),
	})
}

// parseLogLine parses a logrus logfmt line into a LogEntry
func parseLogLine(line string) *LogEntry {
	matches := logPattern.FindStringSubmatch(line)
	if matches != nil {
		t := formatTime(matches[1])
		return &LogEntry{
			Time:    t,
			Type:    normalizeLevel(matches[2]),
			Payload: matches[3],
		}
	}

	// Fallback: try to extract level from unstructured lines
	entry := &LogEntry{
		Time:    "",
		Type:    "info",
		Payload: line,
	}

	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "level=fatal") || strings.Contains(lower, "level=error"):
		entry.Type = "error"
	case strings.Contains(lower, "level=warning") || strings.Contains(lower, "level=warn"):
		entry.Type = "warning"
	case strings.Contains(lower, "level=debug"):
		entry.Type = "debug"
	}

	return entry
}

// normalizeLevel normalizes log level strings
func normalizeLevel(level string) string {
	switch strings.ToLower(level) {
	case "error", "fatal", "panic":
		return "error"
	case "warning", "warn":
		return "warning"
	case "debug":
		return "debug"
	default:
		return "info"
	}
}

// ClearLogs truncates the mihomo log file
func (h *LogHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.pm.ClearLogs(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logs cleared"})
}

// formatTime converts ISO time to a shorter display format
// "2026-06-17T11:00:40.640330002Z" → "11:00:40"
func formatTime(t string) string {
	// Find the time portion after T
	idx := strings.Index(t, "T")
	if idx == -1 {
		return t
	}
	timePart := t[idx+1:]
	// Remove timezone and fractional seconds
	if dotIdx := strings.Index(timePart, "."); dotIdx != -1 {
		timePart = timePart[:dotIdx]
	} else if zIdx := strings.Index(timePart, "Z"); zIdx != -1 {
		timePart = timePart[:zIdx]
	}
	return timePart
}
