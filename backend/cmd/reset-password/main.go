package main

import (
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

type Settings struct {
	PasswordHash string `json:"passwordHash"`
	Theme        string `json:"theme,omitempty"`
	Language     string `json:"language,omitempty"`
	Mihomo       any    `json:"mihomo,omitempty"`
	SubStore     any    `json:"substore,omitempty"`
	Ports        any    `json:"ports,omitempty"`
	Active       string `json:"activeSubscription,omitempty"`
	ExportSubs   bool   `json:"exportIncludeSubscriptions,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: reset-password <new-password>")
		fmt.Println("Example: reset-password mynewpassword123")
		os.Exit(1)
	}

	newPassword := os.Args[1]
	settingsFile := "/data/webui/settings.json"

	// Read current settings
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading settings file: %v\n", err)
		fmt.Println("Please start the container first to initialize the settings.")
		os.Exit(1)
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing settings: %v\n", err)
		os.Exit(1)
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", err)
		os.Exit(1)
	}

	settings.PasswordHash = string(hash)

	// Write back
	updatedData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling settings: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(settingsFile, updatedData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing settings: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password reset successfully!")
	fmt.Println("Please restart the container to apply changes: docker restart <container-name>")
}
