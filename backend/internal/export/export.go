package export

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/zwforum/proxy-web/internal/model"
	"github.com/zwforum/proxy-web/internal/store"
	"github.com/zwforum/proxy-web/internal/substore"
)

// Manifest describes the export package metadata
type Manifest struct {
	Version              string   `json:"version"`
	ExportTime           string   `json:"exportTime"`
	IncludeSubscriptions bool     `json:"includeSubscriptions"`
	ProfileID            string   `json:"profileId"`
	ProfileName          string   `json:"profileName"`
	SubscriptionCount    int      `json:"subscriptionCount"`
}

// Exporter handles profile export/import
type Exporter struct {
	store    *store.FileStore
	subStore *substore.Client
}

func NewExporter(s *store.FileStore, subStoreAddr string) *Exporter {
	return &Exporter{
		store:    s,
		subStore: substore.NewClient(subStoreAddr),
	}
}

// Export creates a zip archive of a profile
func (e *Exporter) Export(profileID string) ([]byte, error) {
	// Load profile registry to find the profile
	registry, err := e.store.LoadProfileRegistry()
	if err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}

	var profile *model.Profile
	for i := range registry.Profiles {
		if registry.Profiles[i].ID == profileID {
			profile = &registry.Profiles[i]
			break
		}
	}
	if profile == nil {
		return nil, fmt.Errorf("profile %s not found", profileID)
	}

	// Read profile data
	rules, _ := e.store.ReadRules(profileID)
	override, _ := e.store.ReadOverride(profileID)

	includeSubs := profile.ExportSettings.IncludeSubscriptions

	// Create zip buffer
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Write manifest
	manifest := Manifest{
		Version:              "1.0",
		ExportTime:           time.Now().UTC().Format(time.RFC3339),
		IncludeSubscriptions: includeSubs,
		ProfileID:            profileID,
		ProfileName:          profile.Name,
	}

	// Fetch subscriptions if needed
	var subs []substore.Subscription
	if includeSubs && profile.SubscriptionName != "" {
		sub, err := e.subStore.GetSubscription(profile.SubscriptionName)
		if err == nil && sub != nil {
			subs = append(subs, *sub)
			manifest.SubscriptionCount = 1
		}
	}

	// Write manifest.json
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	writeZipFile(zw, "manifest.json", manifestData)

	// Write platform data
	profileData, _ := json.MarshalIndent(profile, "", "  ")
	writeZipFile(zw, "platform/meta.json", profileData)
	writeZipFile(zw, "platform/rules.yaml", []byte(rules))
	writeZipFile(zw, "platform/override.yaml", []byte(override))

	// Write subscriptions if included
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
	ProfileID         string `json:"profileId"`
	ProfileName       string `json:"profileName"`
	SubscriptionsImported int `json:"subscriptionsImported"`
}

// Import restores a profile from a zip archive
func (e *Exporter) Import(zipData []byte, forceImportSubs *bool) (*ImportResult, error) {
	// Open zip
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	// Read manifest
	manifestData, err := readZipFile(zr, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	// Read platform data
	profileData, err := readZipFile(zr, "platform/meta.json")
	if err != nil {
		return nil, fmt.Errorf("read profile meta: %w", err)
	}

	var profile model.Profile
	if err := json.Unmarshal(profileData, &profile); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	rules, _ := readZipFile(zr, "platform/rules.yaml")
	override, _ := readZipFile(zr, "platform/override.yaml")

	// Generate new ID to avoid conflicts
	newID := fmt.Sprintf("p-%d", time.Now().UnixNano())
	profile.ID = newID
	profile.UpdatedAt = time.Now()

	// Save profile
	registry, err := e.store.LoadProfileRegistry()
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}
	registry.Profiles = append(registry.Profiles, profile)
	if len(registry.Profiles) == 1 {
		registry.ActiveProfileID = newID
	}
	if err := e.store.SaveProfileRegistry(registry); err != nil {
		return nil, fmt.Errorf("save registry: %w", err)
	}

	// Save rules and override
	e.store.WriteRules(newID, string(rules))
	e.store.WriteOverride(newID, string(override))

	result := &ImportResult{
		ProfileID:   newID,
		ProfileName: profile.Name,
	}

	// Decide whether to import subscriptions
	shouldImportSubs := false
	if forceImportSubs != nil {
		shouldImportSubs = *forceImportSubs
	} else {
		shouldImportSubs = manifest.IncludeSubscriptions
	}

	if shouldImportSubs {
		// Try to read subscription files from zip
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

				// Create subscription in Sub-Store
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
