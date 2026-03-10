package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Report struct {
	MachineID string    `json:"machine_id"`
	Sessions  []Session `json:"sessions"`
	SentAt    time.Time `json:"sent_at"`
}

func sendReport(sessions []Session, cfg *Config) error {
	if len(sessions) == 0 {
		return nil
	}

	report := Report{
		MachineID: machineID(),
		Sessions:  sessions,
		SentAt:    time.Now().UTC(),
	}

	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(cfg.ServerURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	log.Printf("Sent %d session(s) to server", len(sessions))
	return nil
}

// machineID returns a stable anonymous identifier derived from hostname + username.
func machineID() string {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	h := sha256.Sum256([]byte(hostname + username))
	return fmt.Sprintf("%x", h[:8])
}
