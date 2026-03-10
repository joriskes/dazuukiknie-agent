package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DetectedGame struct {
	Name       string
	Source     string // "steam" | "config"
	SteamAppID int
	Process    string
}

var steamCache struct {
	sync.Mutex
	entries map[int]steamCacheEntry
}

type steamCacheEntry struct {
	name      string
	fetchedAt time.Time
}

func init() {
	steamCache.entries = make(map[int]steamCacheEntry)
}

func lookupSteamGame(appID int) (string, error) {
	steamCache.Lock()
	if e, ok := steamCache.entries[appID]; ok && time.Since(e.fetchedAt) < 24*time.Hour {
		steamCache.Unlock()
		return e.name, nil
	}
	steamCache.Unlock()

	url := fmt.Sprintf("https://store.steampowered.com/api/appdetails?appids=%d&filters=basic", appID)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]struct {
		Success bool `json:"success"`
		Data    struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%d", appID)
	entry, ok := result[key]
	if !ok || !entry.Success {
		return "", fmt.Errorf("steam app %d not found", appID)
	}

	name := entry.Data.Name
	steamCache.Lock()
	steamCache.entries[appID] = steamCacheEntry{name: name, fetchedAt: time.Now()}
	steamCache.Unlock()

	return name, nil
}

// Detect returns the currently detected game, or nil if nothing is running.
func Detect(cfg *Config) *DetectedGame {
	// 1. Steam: scan processes for SteamAppId environment variable
	appID, process, err := getSteamRunningApp()
	if err == nil && appID > 0 {
		name, err := lookupSteamGame(appID)
		if err != nil {
			log.Printf("Steam API lookup failed for %d: %v", appID, err)
			name = fmt.Sprintf("Steam App %d", appID)
		}
		return &DetectedGame{
			Name:       name,
			Source:     "steam",
			SteamAppID: appID,
			Process:    process,
		}
	}

	// 2. Active window: match process name against user config
	procName, _, err := getActiveWindowInfo()
	if err != nil || procName == "" {
		return nil
	}

	for _, g := range cfg.Games {
		if strings.EqualFold(g.Process, procName) {
			return &DetectedGame{
				Name:    g.Name,
				Source:  "config",
				Process: procName,
			}
		}
	}

	return nil
}
