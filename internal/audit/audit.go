// Package audit keeps an append-only record of security-relevant actions.
package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	At     time.Time         `json:"at"`
	Actor  string            `json:"actor"`
	Action string            `json:"action"`
	Detail map[string]string `json:"detail,omitempty"`
	IP     string            `json:"ip,omitempty"`
}

type Log struct {
	mu     sync.Mutex
	f      *os.File
	recent []Entry
}

func Open(dir string) (*Log, error) {
	f, err := os.OpenFile(filepath.Join(dir, "audit.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Log{f: f}, nil
}

func (l *Log) Add(actor, action, ip string, detail map[string]string) {
	e := Entry{At: time.Now().UTC(), Actor: actor, Action: action, Detail: detail, IP: ip}
	b, _ := json.Marshal(e)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.f.Write(append(b, '\n'))
	l.recent = append(l.recent, e)
	if len(l.recent) > 500 {
		l.recent = l.recent[len(l.recent)-500:]
	}
}

// Recent returns the newest entries first.
func (l *Log) Recent(n int) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, 0, min(n, len(l.recent)))
	for i := len(l.recent) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, l.recent[i])
	}
	return out
}
