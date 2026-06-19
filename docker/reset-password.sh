#!/bin/sh
set -e

if [ -z "$1" ]; then
  echo "Usage: reset-password <new-password>"
  echo "Example: reset-password mynewpassword123"
  exit 1
fi

NEW_PASSWORD="$1"
SETTINGS_FILE="/data/webui/settings.json"

if [ ! -f "$SETTINGS_FILE" ]; then
  echo "Error: Settings file not found at $SETTINGS_FILE"
  echo "Please start the container first to initialize the settings."
  exit 1
fi

# Use Go to hash the password (since we don't have bcrypt in Alpine by default)
HASHED_PASSWORD=$(cat <<'EOF' | go run -
package main

import (
	"fmt"
	"os"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := os.Args[1]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(hash))
}
EOF
"$NEW_PASSWORD")

if [ -z "$HASHED_PASSWORD" ]; then
  echo "Error: Failed to hash password"
  exit 1
fi

# Update settings.json using sed (since we don't have jq in Alpine by default)
# This is a simple replacement that assumes passwordHash is on its own line
TEMP_FILE=$(mktemp)
awk -v hash="$HASHED_PASSWORD" '
  /"passwordHash":/ {
    print "  \"passwordHash\": \"" hash "\","
    next
  }
  { print }
' "$SETTINGS_FILE" > "$TEMP_FILE"

mv "$TEMP_FILE" "$SETTINGS_FILE"

echo "Password reset successfully!"
echo "Please restart the container to apply changes: docker restart <container-name>"
