// steam.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"golang.org/x/sys/windows/registry"
)

// User struct (Keep as is)
type User struct {
	SteamID     string `json:"steam_id"`
	AccountName string `json:"account_name"`
	PersonaName string `json:"persona_name"`
}

// buildSteamInfo (Keep mostly as is, improve logging/error handling)
func buildSteamInfo() (string, error) {
	steamPath, err := getSteamInstallPath()
	if err != nil {
		return "", fmt.Errorf("could not get Steam path: %w", err)
	}

	loginFileContent, err := readLoginUsersVDF(steamPath)
	if err != nil {
		return "", fmt.Errorf("could not read loginusers.vdf: %w", err)
	}

	users, err := extractUsers(loginFileContent)
	if err != nil {
		// Log the content that failed parsing for debugging
		// log.Printf("Failed to extract users from VDF content:\n%s\n", loginFileContent)
		return "", fmt.Errorf("could not extract users from VDF: %w", err)
	}

	if len(users) == 0 {
		log.Println("No Steam users found in loginusers.vdf")
		// Return empty JSON array instead of erroring
		return "[]", nil
	}

	jsonData, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal Steam users to JSON: %w", err)
	}

	return string(jsonData), nil
}

// getSteamInstallPath (Keep as is)
func getSteamInstallPath() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("unable to open Steam registry key: %w", err)
	}
	defer key.Close()

	steamPath, _, err := key.GetStringValue("SteamPath")
	if err != nil {
		return "", fmt.Errorf("unable to retrieve SteamPath from registry: %w", err)
	}

	// Normalize path separators
	return filepath.Clean(steamPath), nil
}

// readLoginUsersVDF (Keep as is, improve logging)
func readLoginUsersVDF(steamPath string) (string, error) {
	loginUsersVDFPath := filepath.Join(steamPath, "config", "loginusers.vdf")
	log.Println("Reading Steam user file:", loginUsersVDFPath)

	data, err := os.ReadFile(loginUsersVDFPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", loginUsersVDFPath, err)
	}

	return string(data), nil
}

// extractUsers (Keep as is, maybe improve regex slightly)
func extractUsers(fileContent string) ([]User, error) {
	// Regex to find user blocks (slightly more robust with whitespace handling)
	// Using non-greedy matching for names `.+?` might be safer if names contain unexpected characters
	userPattern := `"(?P<SteamID>\d+)"\s*\{\s*"AccountName"\s*"(?P<AccountName>.+?)"\s*"PersonaName"\s*"(?P<PersonaName>.+?)"`

	re, err := regexp.Compile(userPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}

	matches := re.FindAllStringSubmatch(fileContent, -1)
	if matches == nil {
		// This isn't necessarily an error, could be an empty file or no logged-in users
		log.Println("No Steam user matches found in VDF content.")
		return []User{}, nil // Return empty slice, not error
	}
	log.Printf("Found %d potential Steam user entries.\n", len(matches))

	var users []User
	nameMap := make(map[string]int) // Keep track of named capture groups

	// Get mapping from name to index
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			nameMap[name] = i
		}
	}

	for _, match := range matches {
		if len(match) > nameMap["PersonaName"] { // Ensure all expected groups were captured
			user := User{
				SteamID:     match[nameMap["SteamID"]],
				AccountName: match[nameMap["AccountName"]],
				PersonaName: match[nameMap["PersonaName"]],
			}
			users = append(users, user)
		} else {
			log.Println("Warning: Found partial match in VDF, skipping entry.")
		}

	}

	return users, nil
}
