// Package sim is a simulated WD My Cloud EX4 for development and demos.
// Scenarios let you watch MoodNAS (and its eyes) react to failures.
package sim

import (
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

var Scenarios = []string{"normal", "busy", "hot", "disk-failing", "degraded", "ransomware"}

type Sim struct {
	mu       sync.Mutex
	start    time.Time
	scenario string
	fanPct   int
	panel    platform.Panel
	used     float64 // bytes used on md1
	cpuTemp  float64
	readOnly map[string]bool
	files    chan platform.FileEvent
	buttons  chan platform.ButtonEvent
	log      *slog.Logger
}

const tb = 1_000_000_000_000

func New(log *slog.Logger) *Sim {
	s := &Sim{
		start: time.Now(), scenario: "normal", fanPct: 35, used: 5.1 * tb, cpuTemp: 52,
		readOnly: map[string]bool{},
		files:    make(chan platform.FileEvent, 256),
		buttons:  make(chan platform.ButtonEvent, 8),
		log:      log,
	}
	go s.loop()
	return s
}

func (s *Sim) Name() string { return "simulator" }

func (s *Sim) SetScenario(name string) error {
	for _, n := range Scenarios {
		if n == name {
			s.mu.Lock()
			s.scenario = name
			s.mu.Unlock()
			s.log.Info("simulator scenario", "scenario", name)
			return nil
		}
	}
	return fmt.Errorf("unknown scenario %q", name)
}

func (s *Sim) Scenario() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scenario
}

// loop evolves the simulated world once per second.
func (s *Sim) loop() {
	t := time.NewTicker(time.Second)
	for range t.C {
		s.mu.Lock()
		target := 48.0 + float64(s.fanPct)*-0.12 + 8
		switch s.scenario {
		case "busy":
			target += 10
			s.used += 40e6 // ~40 MB/s of writes
		case "hot":
			target += 26
		}
		s.cpuTemp += (target-s.cpuTemp)*0.08 + rand.NormFloat64()*0.15
		s.used += 2e5
		scenario := s.scenario
		s.mu.Unlock()

		if scenario == "ransomware" {
			for i := 0; i < 6; i++ {
				ev := platform.FileEvent{Share: "Family", Kind: "write", Path: fmt.Sprintf("/Family/Photos/IMG_%04d.jpg.locked", rand.IntN(9999)), Entropy: 7.92 + rand.Float64()*0.07, At: time.Now()}
				if i%2 == 1 {
					ev.Kind = "rename"
				}
				s.emit(ev)
			}
		} else if rand.IntN(3) == 0 {
			s.emit(platform.FileEvent{Share: "Documents", Kind: "write", Path: "/Documents/report.odt", Entropy: 4 + rand.Float64()*2, At: time.Now()})
		}
	}
}

func (s *Sim) emit(ev platform.FileEvent) {
	select {
	case s.files <- ev:
	default:
	}
}

func (s *Sim) Sensors() (platform.Sensors, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return platform.Sensors{
		CPUTempC: round1(s.cpuTemp), BoardTempC: round1(s.cpuTemp - 9),
		FanPct: s.fanPct, FanRPM: 600 + s.fanPct*30,
	}, nil
}

func (s *Sim) SetFan(pct int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fanPct = max(0, min(100, pct))
	return nil
}

func (s *Sim) ShowPanel(p platform.Panel) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p != s.panel {
		s.log.Debug("front panel", "mood", p.Mood, "line1", p.Line1, "line2", p.Line2)
	}
	s.panel = p
	return nil
}

func (s *Sim) LastPanel() platform.Panel {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.panel
}

func (s *Sim) Disks() ([]platform.Disk, error) {
	s.mu.Lock()
	scenario := s.scenario
	s.mu.Unlock()
	hours := int(time.Since(s.start).Hours())
	mk := func(bay int, serial string, poh int, temp float64) platform.Disk {
		return platform.Disk{
			Bay: bay, Device: fmt.Sprintf("sd%c", 'a'+bay-1), Model: "WDC WD40EFRX-68N32N0",
			Serial: serial, SizeBytes: 4000787030016, Present: true,
			SMART: &platform.SMART{Healthy: true, TempC: temp, PowerOnHours: poh + hours},
		}
	}
	d := []platform.Disk{
		mk(1, "WD-WCC7K1ABCDE1", 41210, 36),
		mk(2, "WD-WCC7K1ABCDE2", 41208, 38),
		mk(3, "WD-WCC7K1ABCDE3", 39876, 39),
		mk(4, "WD-WCC7K1ABCDE4", 12011, 35),
	}
	switch scenario {
	case "disk-failing":
		sm := d[2].SMART
		sm.Reallocated, sm.Pending, sm.OfflineUncorr, sm.ReportedUncorr = 184, 24, 16, 9
	case "degraded":
		d[2].SMART.Healthy = false
		d[2].SMART.Reallocated, d[2].SMART.Pending = 2104, 312
	case "hot":
		for i := range d {
			d[i].SMART.TempC += 14
		}
	}
	return d, nil
}

func (s *Sim) Arrays() ([]platform.Array, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := platform.Array{
		Name: "md1", Level: "raid5", State: "clean",
		Members: []string{"sda2", "sdb2", "sdc2", "sdd2"}, SizeBytes: 12 * tb,
		UsedBytes: uint64(s.used), Mount: "/shares",
	}
	if s.scenario == "degraded" {
		a.State = "degraded"
		a.Members = []string{"sda2", "sdb2", "sdd2"}
		a.Failed = []string{"sdc2"}
	}
	return []platform.Array{a}, nil
}

func (s *Sim) System() (platform.System, error) {
	s.mu.Lock()
	scenario := s.scenario
	s.mu.Unlock()
	load := 0.35 + rand.Float64()*0.2
	cpu := 6 + rand.Float64()*8
	rx, tx := uint64(2e5+rand.IntN(3e5)), uint64(1e5+rand.IntN(2e5))
	if scenario == "busy" || scenario == "ransomware" {
		load, cpu = 2.6+rand.Float64(), 78+rand.Float64()*15
		rx, tx = uint64(95e6+rand.IntN(10e6)), uint64(3e6)
	}
	return platform.System{
		Model: "WD My Cloud EX4 (simulated)", Kernel: "6.12-moodnas (sim)",
		Uptime: time.Since(s.start), Load1: math.Round(load*100) / 100,
		MemTotal: 512 << 20, MemUsed: uint64(118<<20) + uint64(rand.IntN(20<<20)), CPUPct: math.Round(cpu),
		Net: []platform.NetIf{
			{Name: "eth0", Up: true, Speed: 1000, Address: "192.168.1.50", RxBps: rx, TxBps: tx},
			{Name: "eth1", Up: false},
		},
	}, nil
}

func (s *Sim) FileEvents() <-chan platform.FileEvent { return s.files }
func (s *Sim) Buttons() <-chan platform.ButtonEvent  { return s.buttons }

func (s *Sim) SetReadOnly(share string, ro bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readOnly[share] = ro
	s.log.Warn("share read-only state changed", "share", share, "readOnly", ro)
	return nil
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
