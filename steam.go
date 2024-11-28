package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

type User struct {
	SteamID     string `json:"steam_id"`
	AccountName string `json:"account_name"`
	PersonaName string `json:"persona_name"`
}

func buildSteamInfo() (string, error) {
	// Get the Steam installation path from the registry
	steamPath, err := getSteamInstallPath()
	if err != nil {
		return "", err
	}
	// Read the contents of loginusers.vdf
	loginFile, err := readLoginUsersVDF(steamPath)
	if err != nil {
		return "", err
	}

	users, err := extractUsers(loginFile)

	// Marshal the users slice into JSON
	jsonData, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), err
}

// Function to get the Steam installation path from the registry
func getSteamInstallPath() (string, error) {
	// Open the Steam registry key
	key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("unable to open Steam registry key: %w", err)
	}
	defer key.Close()

	// Get the SteamPath value from the registry
	steamPath, _, err := key.GetStringValue("SteamPath")
	if err != nil {
		return "", fmt.Errorf("unable to retrieve SteamPath from registry: %w", err)
	}

	return steamPath, nil
}

// Function to read the contents of loginusers.vdf as a string
func readLoginUsersVDF(steamPath string) (string, error) {
	// Path to loginusers.vdf file
	loginUsersVDFPath := filepath.Join(steamPath, "config", "loginusers.vdf")

	log.Println("Reading: " + loginUsersVDFPath)

	data, err := os.ReadFile(loginUsersVDFPath) // just pass the file name
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Function to extract users from the provided file
func extractUsers(fileContent string) ([]User, error) {
	userPattern := `(?P<SteamID>\d+)"\s*\{\s*"AccountName"\s*"(?P<AccountName>[^"]+)"\s*"PersonaName"\s*"(?P<PersonaName>[^"]+)"`

	// Compile the regex pattern
	re, err := regexp.Compile(userPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex: %w", err)
	}
	log.Print(fileContent)

	// Find all matches
	matches := re.FindAllStringSubmatch(fileContent, -1)
	if matches == nil {
		return nil, fmt.Errorf("no matches found")
	}
	log.Print("Matches")
	log.Print(matches)

	// Slice to store the extracted users
	var users []User

	// Loop through matches and extract data into the struct
	for _, match := range matches {
		user := User{
			SteamID:     match[1], // Capture group 1 is SteamID
			AccountName: match[2], // Capture group 2 is AccountName
			PersonaName: match[3], // Capture group 3 is PersonaName
		}
		users = append(users, user)
	}

	return users, nil
}
