package local

import (
	"math"
	"sync"
	"time"
)

// Forecast estimates when a volume fills up, by least-squares regression
// over periodic usage samples.
type Forecast struct {
	mu       sync.Mutex
	Interval time.Duration // minimum time between samples
	MinSpan  time.Duration // history needed before predicting
	Max      int
	samples  []sample
}

type sample struct {
	t    time.Time
	used float64
}

type FillPrediction struct {
	Learning     bool    `json:"learning"`
	BytesPerDay  float64 `json:"bytesPerDay"`
	DaysToFull   float64 `json:"daysToFull"` // -1 = not filling
	UsedFraction float64 `json:"usedFraction"`
}

func NewForecast(interval, minSpan time.Duration) *Forecast {
	return &Forecast{Interval: interval, MinSpan: minSpan, Max: 1000}
}

func (f *Forecast) Add(t time.Time, used uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n := len(f.samples); n > 0 && t.Sub(f.samples[n-1].t) < f.Interval {
		return
	}
	f.samples = append(f.samples, sample{t, float64(used)})
	if len(f.samples) > f.Max {
		f.samples = f.samples[len(f.samples)-f.Max:]
	}
}

func (f *Forecast) Predict(used, size uint64) FillPrediction {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := FillPrediction{DaysToFull: -1}
	if size > 0 {
		p.UsedFraction = math.Round(float64(used)/float64(size)*1000) / 1000
	}
	n := len(f.samples)
	if n < 3 || f.samples[n-1].t.Sub(f.samples[0].t) < f.MinSpan {
		p.Learning = true
		return p
	}
	t0 := f.samples[0].t
	var sx, sy, sxx, sxy float64
	for _, s := range f.samples {
		x := s.t.Sub(t0).Hours() / 24
		sx += x
		sy += s.used
		sxx += x * x
		sxy += x * s.used
	}
	fn := float64(n)
	den := fn*sxx - sx*sx
	if den == 0 {
		p.Learning = true
		return p
	}
	slope := (fn*sxy - sx*sy) / den // bytes per day
	p.BytesPerDay = math.Round(slope)
	if slope > 0 && size > used {
		p.DaysToFull = math.Round(float64(size-used)/slope*10) / 10
	}
	return p
}
