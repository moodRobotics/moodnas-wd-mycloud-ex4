// Package history keeps a rolling 24 h record of system metrics for the
// performance charts, in memory, saved to disk now and then so a restart does
// not erase it.
package history

import (
	"encoding/json"
	"math"
	"os"
	"sync"
	"time"
)

// Unknown marks a metric that could not be read (e.g. no temperature sensor).
const Unknown = -1

// Sample is one monitor tick.
type Sample struct {
	T        int64   `json:"t"` // unix seconds
	CPU      float64 `json:"cpu"`
	Load     float64 `json:"load"`
	Mem      float64 `json:"mem"`      // % of RAM used
	CPUTemp  float64 `json:"cpuTemp"`  // °C or Unknown
	DiskTemp float64 `json:"diskTemp"` // hottest disk, °C or Unknown
}

// Point is a (possibly averaged) sample as sent to the UI; nil = no data.
type Point struct {
	T        int64    `json:"t"`
	CPU      *float64 `json:"cpu"`
	Load     *float64 `json:"load"`
	Mem      *float64 `json:"mem"`
	CPUTemp  *float64 `json:"cpuTemp"`
	DiskTemp *float64 `json:"diskTemp"`
}

const Retention = 24 * time.Hour

type Recorder struct {
	mu      sync.Mutex
	path    string
	samples []Sample
	max     int
}

// New keeps up to Retention of samples taken every interval.
func New(path string, interval time.Duration) *Recorder {
	n := int(Retention / max(interval, time.Second))
	r := &Recorder{path: path, max: min(max(n, 360), 43200)}
	r.load()
	return r
}

func (r *Recorder) Add(s Sample) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples = append(r.samples, s)
	if len(r.samples) > r.max {
		r.samples = append(r.samples[:0:0], r.samples[len(r.samples)-r.max:]...)
	}
}

// Query returns the samples newer than `since`, averaged into at most
// maxPoints buckets of equal duration. Empty buckets become gaps.
func (r *Recorder) Query(since time.Time, now time.Time, maxPoints int) []Point {
	r.mu.Lock()
	defer r.mu.Unlock()
	start, end := since.Unix(), now.Unix()
	if end <= start || maxPoints < 1 {
		return []Point{}
	}
	width := float64(end-start) / float64(maxPoints)
	type acc struct {
		n              int
		cpu, load, mem float64
		cpuT, diskT    float64
		nCPUT, nDiskT  int
	}
	buckets := make([]acc, maxPoints)
	for _, s := range r.samples {
		if s.T < start || s.T > end {
			continue
		}
		i := min(int(float64(s.T-start)/width), maxPoints-1)
		b := &buckets[i]
		b.n++
		b.cpu += s.CPU
		b.load += s.Load
		b.mem += s.Mem
		if s.CPUTemp > Unknown {
			b.cpuT += s.CPUTemp
			b.nCPUT++
		}
		if s.DiskTemp > Unknown {
			b.diskT += s.DiskTemp
			b.nDiskT++
		}
	}
	avgD := func(sum float64, n int, scale float64) *float64 {
		if n == 0 {
			return nil
		}
		v := math.Round(sum/float64(n)*scale) / scale
		return &v
	}
	avg := func(sum float64, n int) *float64 { return avgD(sum, n, 10) }
	out := make([]Point, 0, maxPoints)
	for i, b := range buckets {
		p := Point{T: start + int64((float64(i)+0.5)*width)}
		if b.n > 0 {
			p.CPU, p.Load, p.Mem = avg(b.cpu, b.n), avgD(b.load, b.n, 100), avg(b.mem, b.n)
			p.CPUTemp, p.DiskTemp = avg(b.cpuT, b.nCPUT), avg(b.diskT, b.nDiskT)
		}
		out = append(out, p)
	}
	return out
}

// Save writes the samples atomically (called hourly and on shutdown, so the
// disks are not kept awake by frequent writes).
func (r *Recorder) Save() error {
	if r.path == "" {
		return nil
	}
	r.mu.Lock()
	b, err := json.Marshal(r.samples)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func (r *Recorder) load() {
	if r.path == "" {
		return
	}
	b, err := os.ReadFile(r.path)
	if err != nil {
		return
	}
	var s []Sample
	if json.Unmarshal(b, &s) != nil {
		return
	}
	cut := time.Now().Add(-Retention).Unix()
	for _, x := range s {
		if x.T >= cut {
			r.samples = append(r.samples, x)
		}
	}
	if len(r.samples) > r.max {
		r.samples = r.samples[len(r.samples)-r.max:]
	}
}
