// Package mood turns the NAS's condition into one of the Mood Robotics eye
// expressions, with the reasons behind it (as i18n keys + parameters).
package mood

import (
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/local"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

const (
	Calm    = "calm"
	Focused = "focused"
	Sleepy  = "sleepy"
	Worried = "worried"
	Alarmed = "alarmed"
)

type Reason struct {
	Key      string         `json:"key"`
	Params   map[string]any `json:"params,omitempty"`
	Severity int            `json:"severity"` // 0 info, 1 focus, 2 warning, 3 critical
}

type Mood struct {
	State   string   `json:"state"`
	Reasons []Reason `json:"reasons"`
}

type Inputs struct {
	Sensors     platform.Sensors
	MaxDiskTemp float64
	Risks       []*local.DiskRisk
	Arrays      []platform.Array
	Incidents   []local.Incident
	Fill        local.FillPrediction
	CPUPct      float64
	Quiet       bool
}

func Evaluate(in Inputs) Mood {
	var rs []Reason
	add := func(sev int, key string, params map[string]any) {
		rs = append(rs, Reason{Key: key, Params: params, Severity: sev})
	}
	for _, inc := range in.Incidents {
		add(3, "mood.ransomware", map[string]any{"share": inc.Share})
	}
	for _, a := range in.Arrays {
		switch a.State {
		case "degraded", "inactive":
			add(3, "mood.arrayDegraded", map[string]any{"array": a.Name})
		case "recovering", "resyncing":
			add(2, "mood.arrayRebuilding", map[string]any{"array": a.Name, "progress": a.Progress})
		}
	}
	for _, r := range in.Risks {
		if r == nil {
			continue
		}
		p := map[string]any{"bay": r.Bay, "device": r.Device, "pct": int(r.Probability*100 + 0.5)}
		switch r.Level {
		case "high":
			add(3, "mood.diskHighRisk", p)
		case "medium":
			add(2, "mood.diskMediumRisk", p)
		}
	}
	switch {
	case in.Sensors.CPUTempC >= local.CPUCritical || in.MaxDiskTemp >= local.DiskCritical:
		add(3, "mood.tooHot", map[string]any{"cpu": in.Sensors.CPUTempC, "disk": in.MaxDiskTemp})
	case in.Sensors.CPUTempC >= 75 || in.MaxDiskTemp >= 46:
		add(2, "mood.warm", map[string]any{"cpu": in.Sensors.CPUTempC, "disk": in.MaxDiskTemp})
	}
	if !in.Fill.Learning && in.Fill.DaysToFull >= 0 && in.Fill.DaysToFull < 30 {
		add(2, "mood.fillingUp", map[string]any{"days": in.Fill.DaysToFull})
	} else if in.Fill.UsedFraction >= 0.9 {
		add(2, "mood.almostFull", map[string]any{"pct": int(in.Fill.UsedFraction * 100)})
	}
	if in.CPUPct >= 60 {
		add(1, "mood.busy", map[string]any{"cpu": in.CPUPct})
	}

	top := -1
	for _, r := range rs {
		top = max(top, r.Severity)
	}
	m := Mood{Reasons: rs}
	switch top {
	case 3:
		m.State = Alarmed
	case 2:
		m.State = Worried
	case 1:
		m.State = Focused
	default:
		if in.Quiet && in.CPUPct < 20 {
			m.State = Sleepy
			m.Reasons = []Reason{{Key: "mood.resting"}}
		} else {
			m.State = Calm
			m.Reasons = []Reason{{Key: "mood.allGood"}}
		}
	}
	return m
}
