package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type GameEntry struct {
	Process string `json:"process"` // executable name (without .exe on windows)
	Name    string `json:"name"`
}

type Config struct {
	ServerURL string      `json:"server_url"`
	Games     []GameEntry `json:"games"`
}

func defaultConfig() *Config {
	return &Config{
		ServerURL: "https://dazuukiknie.nl/api/sessions",
		Games:     []GameEntry{},
	}
}

func configDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "dazuukiknie")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dazuukiknie")
}

func dataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "dazuukiknie")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "dazuukiknie")
}

func loadConfig() (*Config, error) {
	path := filepath.Join(configDir(), "config.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := defaultConfig()
		_ = saveConfig(cfg)
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}
