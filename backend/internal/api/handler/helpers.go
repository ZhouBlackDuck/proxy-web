package handler

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zwforum/proxy-web/internal/subscription"
)

// resolveSubInput returns the subconverter input: URL for remote, temp file path for local.
func resolveSubInput(sub *subscription.Subscription, tmpDir string) string {
	if sub.Source == "url" && sub.URL != "" {
		return sub.URL
	}
	tmpFile := filepath.Join(tmpDir, sub.Name+".yaml")
	os.WriteFile(tmpFile, []byte(sub.Content), 0644)
	return tmpFile
}

// fixNullProxyGroups replaces "proxy-groups: ~" with "proxy-groups: []" in subconverter output.
func fixNullProxyGroups(yaml string) string {
	yaml = strings.ReplaceAll(yaml, "proxy-groups: ~", "proxy-groups: []")
	yaml = strings.ReplaceAll(yaml, "Proxy Group: ~", "proxy-groups: []")
	yaml = strings.ReplaceAll(yaml, "rules: ~", "rules: []")
	yaml = strings.ReplaceAll(yaml, "Rule: ~", "rules: []")
	return yaml
}
