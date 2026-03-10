package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Game struct {
	Name       string `json:"name"`
	Source     string `json:"source"` // "steam" | "config"
	SteamAppID int    `json:"steam_app_id,omitempty"`
	Process    string `json:"process,omitempty"`
}

type Session struct {
	Game      Game      `json:"game"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Duration  float64   `json:"duration_seconds"`
}

type activeSession struct {
	game      Game
	startedAt time.Time
}

type SessionBuffer struct {
	mu      sync.Mutex
	pending []Session
	active  *activeSession
	path    string
}

func newSessionBuffer() *SessionBuffer {
	dir := dataDir()
	_ = os.MkdirAll(dir, 0755)
	buf := &SessionBuffer{
		path: filepath.Join(dir, "buffer.json"),
	}
	buf.load()
	return buf
}

func (b *SessionBuffer) StartGame(g Game) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active != nil {
		b.finishActive()
	}
	b.active = &activeSession{game: g, startedAt: time.Now()}
}

func (b *SessionBuffer) EndGame() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active != nil {
		b.finishActive()
	}
}

// finishActive must be called with b.mu held.
func (b *SessionBuffer) finishActive() {
	now := time.Now()
	s := Session{
		Game:      b.active.game,
		StartedAt: b.active.startedAt,
		EndedAt:   now,
		Duration:  now.Sub(b.active.startedAt).Seconds(),
	}
	b.active = nil
	// Filter out noise: sessions under 10 seconds
	if s.Duration < 10 {
		return
	}
	b.pending = append(b.pending, s)
	b.save()
	log.Printf("Session recorded: %s (%.0fs)", s.Game.Name, s.Duration)
}

func (b *SessionBuffer) Drain() []Session {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.pending) == 0 {
		return nil
	}
	out := make([]Session, len(b.pending))
	copy(out, b.pending)
	b.pending = b.pending[:0]
	b.save()
	return out
}

func (b *SessionBuffer) HasPending() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending) > 0
}

// Restore puts sessions back if reporting failed.
func (b *SessionBuffer) Restore(sessions []Session) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = append(sessions, b.pending...)
	b.save()
}

func (b *SessionBuffer) save() {
	data, err := json.MarshalIndent(b.pending, "", "  ")
	if err != nil {
		log.Printf("buffer save: %v", err)
		return
	}
	if err := os.WriteFile(b.path, data, 0644); err != nil {
		log.Printf("buffer write: %v", err)
	}
}

func (b *SessionBuffer) load() {
	data, err := os.ReadFile(b.path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Printf("buffer load: %v", err)
		return
	}
	if err := json.Unmarshal(data, &b.pending); err != nil {
		log.Printf("buffer parse: %v", err)
		return
	}
	if len(b.pending) > 0 {
		log.Printf("Loaded %d unsent sessions from disk", len(b.pending))
	}
}
