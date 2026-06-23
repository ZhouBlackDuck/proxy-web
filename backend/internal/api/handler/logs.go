package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zwforum/proxy-web/internal/config"
)

// LogHandler handles log retrieval from server-side storage
type LogHandler struct {
	cfg *config.Config
}

func NewLogHandler(cfg *config.Config) *LogHandler {
	return &LogHandler{cfg: cfg}
}

// GetLogs returns logs from server-side storage (tail-read, last 64KB)
func (h *LogHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	logPath := filepath.Join(h.cfg.DataDir, "mihomo", "mihomo.log")

	const tailSize = 64 * 1024 // 64KB

	f, err := os.Open(logPath)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs": []interface{}{},
		})
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs": []interface{}{},
		})
		return
	}

	// Seek to tail position if file is larger than tailSize
	readOffset := int64(0)
	if info.Size() > tailSize {
		readOffset = info.Size() - tailSize
		if _, err := f.Seek(readOffset, io.SeekStart); err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"logs": []interface{}{},
			})
			return
		}
	}

	buf := make([]byte, info.Size()-readOffset)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"logs": []interface{}{},
		})
		return
	}
	content := string(buf[:n])

	// If we started mid-file, skip the first incomplete line
	if readOffset > 0 {
		if idx := strings.Index(content, "\n"); idx >= 0 {
			content = content[idx+1:]
		}
	}

	// Parse logs
	lines := strings.Split(content, "\n")
	logs := make([]map[string]interface{}, 0)

	for _, line := range lines {
		if line == "" {
			continue
		}

		entry := parseLogLine(line)
		if entry != nil {
			logs = append(logs, entry)
		}
	}

	// Limit to last 500 logs
	if len(logs) > 500 {
		logs = logs[len(logs)-500:]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs": logs,
	})
}

// ClearLogs clears the log file
func (h *LogHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
	logPath := filepath.Join(h.cfg.DataDir, "mihomo", "mihomo.log")
	if err := os.WriteFile(logPath, []byte{}, 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to clear logs",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "logs cleared",
	})
}

func parseLogLine(line string) map[string]interface{} {
	// Parse: time="..." level=... msg="..."
	entry := make(map[string]interface{})

	// Extract time
	if timeMatch := extractField(line, "time"); timeMatch != "" {
		entry["time"] = formatTime(timeMatch)
		entry["timestamp"] = parseTimeToUnix(timeMatch)
	} else {
		entry["time"] = ""
		entry["timestamp"] = 0
	}

	// Extract level
	if levelMatch := extractField(line, "level"); levelMatch != "" {
		entry["type"] = normalizeLevel(levelMatch)
	} else {
		entry["type"] = "info"
	}

	// Extract msg
	if msgMatch := extractField(line, "msg"); msgMatch != "" {
		entry["payload"] = msgMatch
	} else {
		entry["payload"] = line
	}

	return entry
}

func extractField(line, field string) string {
	// Look for field="value" or field=value
	prefix := field + "="
	idx := strings.Index(line, prefix)
	if idx == -1 {
		return ""
	}

	start := idx + len(prefix)
	if start >= len(line) {
		return ""
	}

	// Check if value is quoted
	if line[start] == '"' {
		// Find closing quote
		end := strings.Index(line[start+1:], "\"")
		if end == -1 {
			return ""
		}
		return line[start+1 : start+1+end]
	}

	// Find next space
	end := strings.Index(line[start:], " ")
	if end == -1 {
		return line[start:]
	}
	return line[start : start+end]
}

func formatTime(timeStr string) string {
	// Extract HH:MM:SS from ISO format
	if idx := strings.Index(timeStr, "T"); idx != -1 {
		timePart := timeStr[idx+1:]
		if dotIdx := strings.Index(timePart, "."); dotIdx != -1 {
			return timePart[:dotIdx]
		}
		if zIdx := strings.Index(timePart, "Z"); zIdx != -1 {
			return timePart[:zIdx]
		}
		return timePart
	}
	return timeStr
}

func parseTimeToUnix(timeStr string) int64 {
	// Parse full ISO format: 2026-06-19T05:40:43.123456789Z
	if t, err := time.Parse(time.RFC3339Nano, timeStr); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Unix()
	}
	return time.Now().Unix()
}

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
