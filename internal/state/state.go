// Package state holds MoodNAS's persistent configuration (users, shares, AI
// settings) in a single JSON file written atomically with 0600 permissions.
package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	Name       string    `json:"name"`
	Hash       string    `json:"hash"`
	Role       Role      `json:"role"`
	TOTPSecret string    `json:"totpSecret,omitempty"`
	Created    time.Time `json:"created"`
}

type Share struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ReadOnly    bool     `json:"readOnly"`
	Users       []string `json:"users"` // empty = admins only
	Sentinel    bool     `json:"sentinel"`
}

// AISettings: local features never send data anywhere; remote features use
// the configured provider and are off until an admin enables them.
type AISettings struct {
	Provider string          `json:"provider"` // none | anthropic | ollama
	Model    string          `json:"model"`
	APIKey   string          `json:"apiKey,omitempty"`
	Endpoint string          `json:"endpoint,omitempty"` // ollama base URL
	Features map[string]bool `json:"features"`
}

type FanSettings struct {
	Mode       string `json:"mode"` // ai | manual
	ManualPct  int    `json:"manualPct"`
	QuietStart int    `json:"quietStart"` // hour of day
	QuietEnd   int    `json:"quietEnd"`
}

type Data struct {
	Version  int         `json:"version"`
	Hostname string      `json:"hostname"`
	Language string      `json:"language"`
	Users    []User      `json:"users"`
	Shares   []Share     `json:"shares"`
	AI       AISettings  `json:"ai"`
	Fan      FanSettings `json:"fan"`
}

func defaults() Data {
	return Data{
		Version:  1,
		Hostname: "moodnas",
		Language: "en",
		AI: AISettings{
			Provider: "none",
			Model:    "claude-opus-5-5",
			Features: map[string]bool{
				"diskrisk": true, "sentinel": true, "fan": true, "forecast": true,
				"assistant": false, "explain": false,
			},
		},
		Fan: FanSettings{Mode: "ai", ManualPct: 40, QuietStart: 23, QuietEnd: 7},
	}
}

type Store struct {
	mu   sync.RWMutex
	path string
	data Data
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "state.json"), data: defaults()}
	b, err := os.ReadFile(s.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return s, s.save()
	case err != nil:
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, err
	}
	if s.data.AI.Features == nil {
		s.data.AI.Features = defaults().AI.Features
	}
	return s, nil
}

// Read gives a deep-enough copy of the state to the callback.
func (s *Store) Read() Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clone(s.data)
}

// Update applies fn and persists; if fn returns an error nothing changes.
func (s *Store) Update(fn func(*Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := clone(s.data)
	if err := fn(&next); err != nil {
		return err
	}
	prev := s.data
	s.data = next
	if err := s.save(); err != nil {
		s.data = prev
		return err
	}
	return nil
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func clone(d Data) Data {
	b, _ := json.Marshal(d)
	var out Data
	_ = json.Unmarshal(b, &out)
	return out
}

func (d *Data) FindUser(name string) *User {
	for i := range d.Users {
		if d.Users[i].Name == name {
			return &d.Users[i]
		}
	}
	return nil
}

func (d *Data) FindShare(name string) *Share {
	for i := range d.Shares {
		if d.Shares[i].Name == name {
			return &d.Shares[i]
		}
	}
	return nil
}
