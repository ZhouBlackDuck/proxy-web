package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/zwforum/proxy-web/internal/config"
	"github.com/zwforum/proxy-web/internal/store"
	"github.com/zwforum/proxy-web/internal/substore"
)

// Manifest describes the export package metadata
type Manifest struct {
	Version              string `json:"version"`
	ExportTime           string `json:"exportTime"`
	IncludeSubscriptions bool   `json:"includeSubscriptions"`
	SubscriptionName     string `json:"subscriptionName"`
	SubscriptionCount    int    `json:"subscriptionCount"`
}

// ExportData is the subscription-level data stored in the zip
type ExportData struct {
	Name     string `json:"name"`
	Rules    string `json:"rules"`
	Override string `json:"override"`
}

// Exporter handles subscription export/import
type Exporter struct {
	store    *store.FileStore
	subStore *substore.Client
	cfg      *config.Config
}

func NewExporter(s *store.FileStore, subStoreAddr string, cfg *config.Config) *Exporter {
	return &Exporter{
		store:    s,
		subStore: substore.NewClient(subStoreAddr),
		cfg:      cfg,
	}
}

// ExportSubscription creates a zip archive of a subscription (rules + override + optionally sub data)
// ExportAll exports all platform config + optionally all subscriptions
func (e *Exporter) ExportAll(testSites []map[string]interface{}) ([]byte, error) {
	// Read global rules and override
	globalRules, _ := e.store.ReadSubRules("__global__")
	globalOverride, _ := e.store.ReadSubOverride("__global__")

	includeSubs := e.cfg.ExportIncludeSubscriptions

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	manifest := Manifest{
		Version:              "3.0",
		ExportTime:           time.Now().UTC().Format(time.RFC3339),
		IncludeSubscriptions: includeSubs,
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	writeZipFile(zw, "manifest.json", manifestData)

	// Write platform config (global rules + override + settings + testSites, excluding mihomo/substore)
	settingsMap := map[string]interface{}{
		"theme":                      e.cfg.Theme,
		"language":                   e.cfg.Language,
		"activeSubscription":         e.cfg.ActiveSubscription,
		"exportIncludeSubscriptions": e.cfg.ExportIncludeSubscriptions,
		"ports":                      e.cfg.Ports,
	}
	platformData := map[string]interface{}{
		"globalRules":    globalRules,
		"globalOverride": globalOverride,
		"settings":       settingsMap,
		"testSites":      testSites,
	}
	platformDataJSON, _ := json.MarshalIndent(platformData, "", "  ")
	writeZipFile(zw, "platform/config.json", platformDataJSON)

	// Export all subscriptions if included
	if includeSubs {
		subs, err := e.subStore.ListSubscriptions()
		if err == nil {
			manifest.SubscriptionCount = len(subs)
			// Re-write manifest with count
			zw.Close()
			buf.Reset()
			zw = zip.NewWriter(buf)
			manifestData, _ = json.MarshalIndent(manifest, "", "  ")
			writeZipFile(zw, "manifest.json", manifestData)
			writeZipFile(zw, "platform/config.json", platformDataJSON)

			for _, sub := range subs {
				subData, _ := json.MarshalIndent(sub, "", "  ")
				writeZipFile(zw, fmt.Sprintf("subscriptions/%s.json", sub.Name), subData)
			}
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf.Bytes(), nil
}

func (e *Exporter) ExportSubscription(subName string) ([]byte, error) {
	rules, _ := e.store.ReadSubRules(subName)
	override, _ := e.store.ReadSubOverride(subName)

	includeSubs := e.cfg.ExportIncludeSubscriptions

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	manifest := Manifest{
		Version:              "2.0",
		ExportTime:           time.Now().UTC().Format(time.RFC3339),
		IncludeSubscriptions: includeSubs,
		SubscriptionName:     subName,
	}

	// Fetch subscription data from Sub-Store if needed
	var subs []substore.Subscription
	if includeSubs && subName != "" {
		sub, err := e.subStore.GetSubscription(subName)
		if err == nil && sub != nil {
			subs = append(subs, *sub)
			manifest.SubscriptionCount = 1
		}
	}

	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	writeZipFile(zw, "manifest.json", manifestData)

	// Write subscription config data (rules + override)
	exportData := ExportData{
		Name:     subName,
		Rules:    rules,
		Override: override,
	}
	exportDataJSON, _ := json.MarshalIndent(exportData, "", "  ")
	writeZipFile(zw, "subscription/data.json", exportDataJSON)

	// Write Sub-Store subscription data if included
	if includeSubs && len(subs) > 0 {
		for _, sub := range subs {
			subData, _ := json.MarshalIndent(sub, "", "  ")
			writeZipFile(zw, fmt.Sprintf("subscriptions/%s.json", sub.Name), subData)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// ImportResult describes what was imported
type ImportResult struct {
	SubscriptionName    string                   `json:"subscriptionName"`
	SubscriptionsImported int                    `json:"subscriptionsImported"`
	TestSites           []map[string]interface{} `json:"testSites,omitempty"`
}

// Import restores subscription data from a zip archive
func (e *Exporter) Import(zipData []byte, forceImportSubs *bool) (*ImportResult, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	manifestData, err := readZipFile(zr, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	result := &ImportResult{
		SubscriptionName: manifest.SubscriptionName,
	}

	// Read subscription data (rules + override)
	exportDataJSON, err := readZipFile(zr, "subscription/data.json")
	if err == nil {
		var exportData ExportData
		if err := json.Unmarshal(exportDataJSON, &exportData); err == nil {
			subName := exportData.Name
			if subName == "" {
				subName = manifest.SubscriptionName
			}
			if exportData.Rules != "" {
				e.store.WriteSubRules(subName, exportData.Rules)
			}
			if exportData.Override != "" {
				e.store.WriteSubOverride(subName, exportData.Override)
			}
			result.SubscriptionName = subName
		}
	}

	// Read platform config (global rules + override + settings + testSites, excluding mihomo/substore)
	platformDataJSON, err := readZipFile(zr, "platform/config.json")
	if err == nil {
		var platformData struct {
			GlobalRules    string                   `json:"globalRules"`
			GlobalOverride string                   `json:"globalOverride"`
			Settings       map[string]interface{} `json:"settings"`
			TestSites      []map[string]interface{} `json:"testSites"`
		}
		if err := json.Unmarshal(platformDataJSON, &platformData); err == nil {
			if platformData.GlobalRules != "" {
				e.store.WriteSubRules("__global__", platformData.GlobalRules)
			}
			if platformData.GlobalOverride != "" {
				e.store.WriteSubOverride("__global__", platformData.GlobalOverride)
			}
			// Import settings but skip mihomo/substore
			if platformData.Settings != nil {
				delete(platformData.Settings, "mihomo")
				delete(platformData.Settings, "substore")
				// Merge into current config
				if data, err := json.Marshal(platformData.Settings); err == nil {
					json.Unmarshal(data, e.cfg)
					e.cfg.Save()
				}
			}
			// Extract testSites
			if platformData.TestSites != nil {
				result.TestSites = platformData.TestSites
			}
		}
	}

	// Decide whether to import Sub-Store subscriptions
	shouldImportSubs := false
	if forceImportSubs != nil {
		shouldImportSubs = *forceImportSubs
	} else {
		shouldImportSubs = manifest.IncludeSubscriptions
	}

	if shouldImportSubs {
		for _, f := range zr.File {
			if len(f.Name) > 14 && f.Name[:14] == "subscriptions/" {
				rc, err := f.Open()
				if err != nil {
					continue
				}
				data, _ := io.ReadAll(rc)
				rc.Close()

				var sub substore.Subscription
				if err := json.Unmarshal(data, &sub); err != nil {
					continue
				}

				if err := e.subStore.CreateSubscription(sub); err == nil {
					result.SubscriptionsImported++
				}
			}
		}
	}

	return result, nil
}

// --- Zip helpers ---

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readZipFile(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found in zip: %s", name)
}
