package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/zwforum/proxy-web/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: reset-password <new-password>")
		fmt.Println("Example: reset-password mynewpassword123")
		os.Exit(1)
	}

	newPassword := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg.PasswordHash == "" {
		fmt.Println("No password is currently set. Please start the container first to initialize.")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", err)
		os.Exit(1)
	}

	cfg.PasswordHash = string(hash)
	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Password reset successfully!")
	fmt.Println("Please restart the container to apply changes: docker restart <container-name>")
}
