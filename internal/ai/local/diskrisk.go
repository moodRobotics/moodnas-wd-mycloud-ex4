// Package local contains the AI features that run entirely on the NAS.
// Nothing here sends data anywhere.
package local

import (
	"encoding/json"
	"math"
	"os"
	"sort"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

// DiskModel is a logistic-regression failure-risk model over SMART
// attributes known to predict failure (Backblaze, Pinheiro et al. 2007).
// The built-in weights are hand-set priors; tools/ai/train_diskrisk.py fits
// real weights from Backblaze's public drive-stats data and writes a JSON
// file that LoadDiskModel picks up.
type DiskModel struct {
	Bias    float64            `json:"bias"`
	Weights map[string]float64 `json:"weights"`
	Source  string             `json:"source"`
}

var DefaultDiskModel = DiskModel{
	Bias: -6.0,
	Weights: map[string]float64{
		"reallocated": 0.55, "reported_uncorrectable": 0.70, "command_timeout": 0.25,
		"pending": 0.90, "offline_uncorrectable": 0.80, "age_years": 0.12,
		"temp_over_45": 0.05, "smart_failed": 3.0,
	},
	Source: "built-in priors",
}

func LoadDiskModel(path string) DiskModel {
	b, err := os.ReadFile(path)
	if err != nil {
		return DefaultDiskModel
	}
	var m DiskModel
	if json.Unmarshal(b, &m) != nil || m.Weights == nil {
		return DefaultDiskModel
	}
	return m
}

type Factor struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Weight float64 `json:"contribution"`
}

type DiskRisk struct {
	Device      string   `json:"device"`
	Bay         int      `json:"bay"`
	Probability float64  `json:"probability"` // 0..1
	Level       string   `json:"level"`       // low | medium | high
	Factors     []Factor `json:"factors"`     // biggest contributors first
}

func features(s *platform.SMART) map[string]float64 {
	l := func(v int) float64 { return math.Log1p(float64(max(v, 0))) }
	f := map[string]float64{
		"reallocated":            l(s.Reallocated),
		"reported_uncorrectable": l(s.ReportedUncorr),
		"command_timeout":        l(s.CommandTimeout),
		"pending":                l(s.Pending),
		"offline_uncorrectable":  l(s.OfflineUncorr),
		"age_years":              float64(s.PowerOnHours) / 8766,
		"temp_over_45":           math.Max(0, s.TempC-45),
	}
	if !s.Healthy {
		f["smart_failed"] = 1
	}
	return f
}

func (m DiskModel) Score(d platform.Disk) *DiskRisk {
	if d.SMART == nil {
		return nil
	}
	z := m.Bias
	var factors []Factor
	for name, v := range features(d.SMART) {
		c := m.Weights[name] * v
		z += c
		if c > 0.05 {
			factors = append(factors, Factor{Name: name, Value: v, Weight: math.Round(c*100) / 100})
		}
	}
	sort.Slice(factors, func(i, j int) bool { return factors[i].Weight > factors[j].Weight })
	p := 1 / (1 + math.Exp(-z))
	r := &DiskRisk{Device: d.Device, Bay: d.Bay, Probability: math.Round(p*1000) / 1000, Factors: factors}
	switch {
	case p >= 0.25:
		r.Level = "high"
	case p >= 0.05:
		r.Level = "medium"
	default:
		r.Level = "low"
	}
	return r
}
