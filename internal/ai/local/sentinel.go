package local

import (
	"math"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

// Entropy returns the Shannon entropy of b in bits per byte (0..8).
// Encrypted data sits near 8; documents and text far below.
func Entropy(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var counts [256]int
	for _, c := range b {
		counts[c]++
	}
	n := float64(len(b))
	h := 0.0
	for _, c := range counts {
		if c > 0 {
			p := float64(c) / n
			h -= p * math.Log2(p)
		}
	}
	return h
}

// Formats that are high-entropy by nature, so a high-entropy write to them is
// expected and weighs little.
var compressedExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".heic": true,
	".mp4": true, ".mkv": true, ".mov": true, ".avi": true, ".mp3": true, ".flac": true, ".aac": true,
	".zip": true, ".gz": true, ".7z": true, ".rar": true, ".xz": true, ".zst": true, ".bz2": true,
	".pdf": true, ".docx": true, ".xlsx": true, ".pptx": true, ".odt": true, ".ods": true, ".iso": true,
}

// Incident is raised when a share shows a ransomware-like write pattern.
type Incident struct {
	Share    string    `json:"share"`
	At       time.Time `json:"at"`
	Score    float64   `json:"score"`
	Events   int       `json:"events"`
	Sample   []string  `json:"sample"`
	Frozen   bool      `json:"frozen"`
	Resolved bool      `json:"resolved"`
}

// Sentinel scores file activity per share over a sliding window. Signals:
// many high-entropy writes to files whose names don't suggest compressed
// content, renames to unknown extensions (".jpg.locked"), and mass deletes.
type Sentinel struct {
	mu        sync.Mutex
	Window    time.Duration
	Threshold float64
	events    map[string][]scored
	incidents []Incident
}

type scored struct {
	at    time.Time
	score float64
	path  string
}

func NewSentinel() *Sentinel {
	return &Sentinel{Window: time.Minute, Threshold: 30, events: map[string][]scored{}}
}

func eventScore(ev platform.FileEvent) float64 {
	ext := strings.ToLower(path.Ext(ev.Path))
	base := strings.ToLower(path.Base(ev.Path))
	// double extension such as photo.jpg.locked: inner ext known, outer unknown
	inner := strings.ToLower(path.Ext(strings.TrimSuffix(base, ext)))
	doubleExt := inner != "" && compressedExt[inner] && !compressedExt[ext]
	switch ev.Kind {
	case "write":
		if ev.Entropy < 7.5 {
			return 0
		}
		if compressedExt[ext] {
			// a fast photo import is ~10 files/s: needs >1500/min to matter
			return 0.02
		}
		if doubleExt {
			return 1.5
		}
		return 1
	case "rename":
		if doubleExt || (ext != "" && !compressedExt[ext] && len(ext) > 5) {
			return 2
		}
		return 0.1
	case "delete":
		return 0.5
	}
	return 0
}

// Observe records an event; it returns a new incident when the share's score
// crosses the threshold (one incident per share until resolved).
func (s *Sentinel) Observe(ev platform.FileEvent) *Incident {
	sc := eventScore(ev)
	if sc == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cut := ev.At.Add(-s.Window)
	list := s.events[ev.Share]
	i := 0
	for i < len(list) && list[i].at.Before(cut) {
		i++
	}
	list = append(list[i:], scored{ev.At, sc, ev.Path})
	s.events[ev.Share] = list
	total := 0.0
	for _, e := range list {
		total += e.score
	}
	if total < s.Threshold {
		return nil
	}
	for _, inc := range s.incidents {
		if inc.Share == ev.Share && !inc.Resolved {
			return nil
		}
	}
	inc := Incident{Share: ev.Share, At: ev.At, Score: math.Round(total*10) / 10, Events: len(list)}
	for j := len(list) - 1; j >= 0 && len(inc.Sample) < 5; j-- {
		inc.Sample = append(inc.Sample, list[j].path)
	}
	s.incidents = append(s.incidents, inc)
	return &s.incidents[len(s.incidents)-1]
}

func (s *Sentinel) MarkFrozen(share string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.incidents {
		if s.incidents[i].Share == share && !s.incidents[i].Resolved {
			s.incidents[i].Frozen = true
		}
	}
}

func (s *Sentinel) Resolve(share string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.incidents {
		if s.incidents[i].Share == share {
			s.incidents[i].Resolved = true
		}
	}
	delete(s.events, share)
}

func (s *Sentinel) Incidents() []Incident {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Incident{}, s.incidents...)
}

// Active never returns nil, so the API always serialises a JSON array.
func (s *Sentinel) Active() []Incident {
	out := []Incident{}
	for _, i := range s.Incidents() {
		if !i.Resolved {
			out = append(out, i)
		}
	}
	return out
}
