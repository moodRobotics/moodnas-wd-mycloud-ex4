// Package monitor is MoodNAS's heartbeat: it samples the hardware, runs the
// local AI models, drives the fan and front panel, and keeps the latest
// snapshot for the API and the assistant.
package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/ai/local"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/audit"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/mood"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/state"
)

type Snapshot struct {
	At        time.Time            `json:"at"`
	Platform  string               `json:"platform"`
	Mood      mood.Mood            `json:"mood"`
	Sensors   platform.Sensors     `json:"sensors"`
	Fan       local.FanDecision    `json:"fan"`
	System    platform.System      `json:"system"`
	Disks     []platform.Disk      `json:"disks"`
	Risks     []*local.DiskRisk    `json:"risks"`
	Arrays    []platform.Array     `json:"arrays"`
	Fill      local.FillPrediction `json:"fill"`
	Incidents []local.Incident     `json:"incidents"`
	Panel     platform.Panel       `json:"panel"`
}

type Monitor struct {
	P        platform.Platform
	Store    *state.Store
	Audit    *audit.Log
	Log      *slog.Logger
	Interval time.Duration

	DiskModel local.DiskModel
	Sentinel  *local.Sentinel
	Forecast  *local.Forecast
	Fan       *local.FanController

	mu        sync.RWMutex
	snap      Snapshot
	setupCode string
}

func (m *Monitor) SetSetupCode(c string) {
	m.mu.Lock()
	m.setupCode = c
	m.mu.Unlock()
}

func (m *Monitor) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snap
}

func (m *Monitor) Run(ctx context.Context) {
	go m.watchFiles(ctx)
	go m.watchButtons(ctx)
	m.tick()
	t := time.NewTicker(m.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.tick()
		}
	}
}

func (m *Monitor) Tick() { m.tick() }

func (m *Monitor) tick() {
	now := time.Now()
	cfg := m.Store.Read()
	s := Snapshot{At: now, Platform: m.P.Name()}
	var err error
	if s.Sensors, err = m.P.Sensors(); err != nil {
		m.Log.Warn("sensors", "err", err)
	}
	if s.System, err = m.P.System(); err != nil {
		m.Log.Warn("system", "err", err)
	}
	if s.Disks, err = m.P.Disks(); err != nil {
		m.Log.Warn("disks", "err", err)
	}
	if s.Arrays, err = m.P.Arrays(); err != nil {
		m.Log.Warn("arrays", "err", err)
	}

	// empty lists, not null, in the JSON API
	s.Risks, s.Disks, s.Arrays = append([]*local.DiskRisk{}, s.Risks...), append([]platform.Disk{}, s.Disks...), append([]platform.Array{}, s.Arrays...)
	maxDisk := 0.0
	for _, d := range s.Disks {
		if d.SMART != nil {
			maxDisk = max(maxDisk, d.SMART.TempC)
			if cfg.AI.Features["diskrisk"] {
				s.Risks = append(s.Risks, m.DiskModel.Score(d))
			}
		}
	}

	var used, size uint64
	for _, a := range s.Arrays {
		used, size = used+a.UsedBytes, size+a.SizeBytes
	}
	if cfg.AI.Features["forecast"] {
		m.Forecast.Add(now, used)
		s.Fill = m.Forecast.Predict(used, size)
	} else {
		s.Fill = local.FillPrediction{Learning: true, DaysToFull: -1}
	}

	// Fan: predictive AI policy, or the admin's fixed value. Critical
	// temperatures always force 100% regardless of mode.
	m.Fan.QuietStart, m.Fan.QuietEnd = cfg.Fan.QuietStart, cfg.Fan.QuietEnd
	s.Fan = m.Fan.Decide(now, s.Sensors.CPUTempC, maxDisk)
	target := s.Fan.Pct
	if cfg.Fan.Mode == "manual" || !cfg.AI.Features["fan"] {
		target = cfg.Fan.ManualPct
		if s.Fan.Critical {
			target = 100
		}
		s.Fan.Pct, s.Fan.Reason = target, "fan.reason.manual"
		if s.Fan.Critical {
			s.Fan.Reason = "fan.reason.critical"
		}
	}
	if err := m.P.SetFan(target); err != nil && !errors.Is(err, platform.ErrNotSupported) {
		m.Log.Warn("set fan", "err", err)
	}
	if s2, err := m.P.Sensors(); err == nil {
		s.Sensors = s2
	}

	s.Incidents = m.Sentinel.Active()
	s.Mood = mood.Evaluate(mood.Inputs{
		Sensors: s.Sensors, MaxDiskTemp: maxDisk, Risks: s.Risks, Arrays: s.Arrays,
		Incidents: s.Incidents, Fill: s.Fill, CPUPct: s.System.CPUPct, Quiet: s.Fan.Quiet,
	})

	m.mu.RLock()
	code := m.setupCode
	m.mu.RUnlock()
	s.Panel = panelFor(cfg, s, code, maxDisk)
	if err := m.P.ShowPanel(s.Panel); err != nil && !errors.Is(err, platform.ErrNotSupported) {
		m.Log.Warn("panel", "err", err)
	}

	m.mu.Lock()
	m.snap = s
	m.mu.Unlock()
}

// panelFor composes the two text lines shown next to the eyes on the OLED.
// The panel is tiny and language-neutral on purpose (numbers + IP).
func panelFor(cfg state.Data, s Snapshot, setupCode string, maxDisk float64) platform.Panel {
	p := platform.Panel{Mood: s.Mood.State, Line1: strings.ToUpper(cfg.Hostname)}
	for _, n := range s.System.Net {
		if n.Up && n.Address != "" {
			p.Line2 = n.Address
			break
		}
	}
	if setupCode != "" {
		p.Line1, p.Line2 = "SETUP CODE", setupCode
		return p
	}
	if len(s.Mood.Reasons) > 0 && s.Mood.Reasons[0].Severity >= 2 {
		r := s.Mood.Reasons[0]
		switch r.Key {
		case "mood.ransomware":
			p.Line2 = fmt.Sprintf("! SHARE FROZEN: %v", r.Params["share"])
		case "mood.diskHighRisk", "mood.diskMediumRisk":
			p.Line2 = fmt.Sprintf("! CHECK DISK %v", r.Params["bay"])
		case "mood.arrayDegraded":
			p.Line2 = "! RAID DEGRADED"
		case "mood.tooHot", "mood.warm":
			// the hottest real reading, never the prediction
			p.Line2 = fmt.Sprintf("! TEMP %.0fC", max(s.Sensors.CPUTempC, maxDisk))
		}
	}
	return p
}

func (m *Monitor) watchFiles(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-m.P.FileEvents():
			if !m.Store.Read().AI.Features["sentinel"] {
				continue
			}
			inc := m.Sentinel.Observe(ev)
			if inc == nil {
				continue
			}
			m.Log.Error("ransomware-like activity detected", "share", inc.Share, "score", inc.Score)
			m.Audit.Add("sentinel", "sentinel.incident", "", map[string]string{"share": inc.Share, "score": fmt.Sprint(inc.Score)})
			cfg := m.Store.Read()
			sh := cfg.FindShare(inc.Share)
			if sh == nil || sh.Sentinel {
				if err := m.P.SetReadOnly(inc.Share, true); err == nil {
					m.Sentinel.MarkFrozen(inc.Share)
					m.Audit.Add("sentinel", "share.frozen", "", map[string]string{"share": inc.Share})
				}
			}
			m.tick()
		}
	}
}

func (m *Monitor) watchButtons(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case b := <-m.P.Buttons():
			m.Log.Info("button", "name", b.Name)
			m.Audit.Add("hardware", "button."+b.Name, "", nil)
		}
	}
}

// ResolveIncident unfreezes a share after an admin has reviewed it.
func (m *Monitor) ResolveIncident(share string) error {
	if err := m.P.SetReadOnly(share, false); err != nil && !errors.Is(err, platform.ErrNotSupported) {
		return err
	}
	m.Sentinel.Resolve(share)
	m.tick()
	return nil
}
