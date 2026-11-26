package local

import (
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

func TestDiskRisk(t *testing.T) {
	healthy := platform.Disk{Device: "sda", SMART: &platform.SMART{Healthy: true, TempC: 36, PowerOnHours: 40000}}
	failing := platform.Disk{Device: "sdc", SMART: &platform.SMART{Healthy: true, TempC: 39, PowerOnHours: 40000,
		Reallocated: 184, Pending: 24, OfflineUncorr: 16, ReportedUncorr: 9}}
	h, f := DefaultDiskModel.Score(healthy), DefaultDiskModel.Score(failing)
	if h.Level != "low" || h.Probability > 0.02 {
		t.Fatalf("healthy disk scored %+v", h)
	}
	if f.Level != "high" || f.Probability < 0.9 {
		t.Fatalf("failing disk scored %+v", f)
	}
	if f.Factors[0].Name != "pending" {
		t.Fatalf("top factor should be pending sectors, got %+v", f.Factors)
	}
	if DefaultDiskModel.Score(platform.Disk{}) != nil {
		t.Fatal("disk without SMART must not be scored")
	}
}

func TestEntropy(t *testing.T) {
	if e := Entropy([]byte(strings.Repeat("hello world ", 500))); e > 4 {
		t.Fatalf("text entropy too high: %v", e)
	}
	b := make([]byte, 64<<10)
	rand.Read(b)
	if e := Entropy(b); e < 7.9 {
		t.Fatalf("random entropy too low: %v", e)
	}
}

func TestSentinel(t *testing.T) {
	s := NewSentinel()
	now := time.Unix(1_700_000_000, 0)
	// ordinary activity: photos (high entropy but compressed type) and documents
	for i := 0; i < 200; i++ {
		ev := platform.FileEvent{Share: "Family", Kind: "write", Path: "/Family/IMG_1.jpg", Entropy: 7.95, At: now.Add(time.Duration(i) * 100 * time.Millisecond)}
		if s.Observe(ev) != nil {
			t.Fatal("false positive on normal photo uploads")
		}
	}
	// ransomware: encrypted rewrites + renames to .locked
	var inc *Incident
	for i := 0; i < 40 && inc == nil; i++ {
		at := now.Add(30*time.Second + time.Duration(i)*100*time.Millisecond)
		inc = s.Observe(platform.FileEvent{Share: "Docs", Kind: "write", Path: "/Docs/a.docx.locked", Entropy: 7.97, At: at})
		if inc == nil {
			inc = s.Observe(platform.FileEvent{Share: "Docs", Kind: "rename", Path: "/Docs/b.jpg.locked", At: at})
		}
	}
	if inc == nil || inc.Share != "Docs" {
		t.Fatalf("ransomware pattern not detected: %+v", inc)
	}
	if len(s.Active()) != 1 {
		t.Fatal("expected exactly one active incident")
	}
	s.Resolve("Docs")
	if len(s.Active()) != 0 {
		t.Fatal("resolve failed")
	}
}

func TestForecast(t *testing.T) {
	f := NewForecast(time.Minute, time.Hour)
	t0 := time.Unix(0, 0)
	const perDay = 10e9
	for i := 0; i <= 48; i++ {
		ts := t0.Add(time.Duration(i) * time.Hour)
		f.Add(ts, uint64(1e12+perDay*ts.Sub(t0).Hours()/24))
	}
	p := f.Predict(uint64(1e12+2*perDay), 2e12)
	if p.Learning || p.BytesPerDay < 9.9e9 || p.BytesPerDay > 10.1e9 {
		t.Fatalf("bad slope %+v", p)
	}
	if p.DaysToFull < 97 || p.DaysToFull > 99 {
		t.Fatalf("bad days to full %+v", p)
	}
	if NewForecast(time.Minute, time.Hour).Predict(1, 2).Learning != true {
		t.Fatal("empty forecast must be learning")
	}
}

func TestFanController(t *testing.T) {
	f := NewFanController(23, 7)
	noon := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	d := f.Decide(noon, 50, 34)
	if d.Pct != 20 || d.Critical {
		t.Fatalf("idle decision %+v", d)
	}
	// rising temperature: predictive control should spin up before reaching 70
	var last FanDecision
	for i := 0; i < 6; i++ {
		last = f.Decide(noon.Add(time.Duration(i+1)*30*time.Second), 60+float64(i)*1.5, 36)
	}
	if last.Reason != "fan.reason.rising" || last.PredictedCPU <= 67.5 || last.Pct < 55 {
		t.Fatalf("predictive decision %+v", last)
	}
	// night: quiet cap
	nf := NewFanController(23, 7)
	n := nf.Decide(time.Date(2026, 1, 1, 2, 0, 0, 0, time.UTC), 76, 44)
	if !n.Quiet || n.Pct > 45 {
		t.Fatalf("quiet decision %+v", n)
	}
	// safety beats silence
	c := nf.Decide(time.Date(2026, 1, 1, 2, 1, 0, 0, time.UTC), 70, 56)
	if !c.Critical || c.Pct != 100 {
		t.Fatalf("critical decision %+v", c)
	}
}

// SMART values read from a real EX4 (Stage 0, 2026-11-25): four ~7-year-old
// WD20EFRX, the one in bay 3 with 125 reallocated sectors while its RAID
// rebuilt. The model must single it out.
func TestDiskRiskRealEX4(t *testing.T) {
	healthy := platform.Disk{Device: "sda", Bay: 1, SMART: &platform.SMART{Healthy: true, TempC: 39, PowerOnHours: 60990}}
	bay3 := platform.Disk{Device: "sdc", Bay: 3, SMART: &platform.SMART{Healthy: true, TempC: 42, PowerOnHours: 59796, Reallocated: 125}}
	h, r := DefaultDiskModel.Score(healthy), DefaultDiskModel.Score(bay3)
	if h.Level != "low" {
		t.Fatalf("healthy old disk scored %+v", h)
	}
	if r.Level != "medium" || r.Factors[0].Name != "reallocated" {
		t.Fatalf("bay 3 scored %+v", r)
	}
	t.Logf("healthy %.1f%%, bay 3 %.1f%%", h.Probability*100, r.Probability*100)
}
