package local

import (
	"math"
	"sync"
	"time"
)

// FanController is a predictive fan policy: it extrapolates the temperature
// trend a few minutes ahead and spins up before the heat arrives, keeps the
// NAS quiet at night, and never trades safety for silence.
type FanController struct {
	mu         sync.Mutex
	history    []tempSample
	last       int
	Lookahead  time.Duration
	QuietStart int
	QuietEnd   int
	QuietCap   int
}

type tempSample struct {
	t   time.Time
	cpu float64
}

type FanDecision struct {
	Pct          int     `json:"pct"`
	PredictedCPU float64 `json:"predictedCpuC"`
	TrendPerMin  float64 `json:"trendPerMin"`
	Quiet        bool    `json:"quiet"`
	Reason       string  `json:"reason"` // i18n key
	Critical     bool    `json:"critical"`
}

const (
	CPUCritical  = 90.0
	DiskCritical = 55.0
)

func NewFanController(quietStart, quietEnd int) *FanController {
	return &FanController{Lookahead: 3 * time.Minute, QuietStart: quietStart, QuietEnd: quietEnd, QuietCap: 45, last: -1}
}

// ramp maps x linearly from (x0 -> y0) to (x1 -> y1), clamped.
func ramp(x, x0, x1, y0, y1 float64) float64 {
	if x <= x0 {
		return y0
	}
	if x >= x1 {
		return y1
	}
	return y0 + (x-x0)/(x1-x0)*(y1-y0)
}

func inQuiet(h, start, end int) bool {
	if start == end {
		return false
	}
	if start < end {
		return h >= start && h < end
	}
	return h >= start || h < end
}

func (f *FanController) Decide(now time.Time, cpu, maxDisk float64) FanDecision {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.history = append(f.history, tempSample{now, cpu})
	cut := now.Add(-5 * time.Minute)
	for len(f.history) > 2 && f.history[0].t.Before(cut) {
		f.history = f.history[1:]
	}
	trend := 0.0 // °C per minute
	if n := len(f.history); n >= 3 {
		t0 := f.history[0].t
		var sx, sy, sxx, sxy float64
		for _, s := range f.history {
			x := s.t.Sub(t0).Minutes()
			sx, sy, sxx, sxy = sx+x, sy+s.cpu, sxx+x*x, sxy+x*s.cpu
		}
		if den := float64(n)*sxx - sx*sx; den > 1e-9 {
			trend = (float64(n)*sxy - sx*sy) / den
		}
	}
	predicted := cpu + math.Max(0, trend)*f.Lookahead.Minutes()
	d := FanDecision{PredictedCPU: math.Round(predicted*10) / 10, TrendPerMin: math.Round(trend*100) / 100, Reason: "fan.reason.normal"}

	pct := math.Max(ramp(predicted, 55, 85, 20, 100), ramp(maxDisk, 35, 50, 20, 100))
	if trend > 0.5 {
		d.Reason = "fan.reason.rising"
	}
	if inQuiet(now.Hour(), f.QuietStart, f.QuietEnd) && cpu < 78 && maxDisk < 46 {
		d.Quiet = true
		if pct > float64(f.QuietCap) {
			pct = float64(f.QuietCap)
		}
		d.Reason = "fan.reason.quiet"
	}
	if cpu >= CPUCritical || maxDisk >= DiskCritical {
		pct, d.Critical, d.Reason = 100, true, "fan.reason.critical"
	}
	p := int(math.Round(pct))
	// hysteresis: ignore changes under 5 points unless going to 100
	if f.last >= 0 && p != 100 && abs(p-f.last) < 5 {
		p = f.last
	}
	f.last = p
	d.Pct = p
	return d
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
