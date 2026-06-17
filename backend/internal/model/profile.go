package model

import "time"

// Profile represents a configuration profile
type Profile struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	SubscriptionName string         `json:"subscriptionName,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	ExportSettings   ExportSettings `json:"exportSettings"`
}

// ExportSettings controls what gets included in profile export
type ExportSettings struct {
	IncludeSubscriptions bool `json:"includeSubscriptions"`
}

// ProfileRegistry holds all profiles and the active profile ID
type ProfileRegistry struct {
	ActiveProfileID string    `json:"activeProfileId,omitempty"`
	Profiles        []Profile `json:"profiles"`
}

// CreateProfileRequest is the request body for creating a new profile
type CreateProfileRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	SubscriptionName string `json:"subscriptionName,omitempty"`
}

// UpdateProfileRequest is the request body for updating a profile
type UpdateProfileRequest struct {
	Name             *string         `json:"name,omitempty"`
	Description      *string         `json:"description,omitempty"`
	SubscriptionName *string         `json:"subscriptionName,omitempty"`
	ExportSettings   *ExportSettings `json:"exportSettings,omitempty"`
}
