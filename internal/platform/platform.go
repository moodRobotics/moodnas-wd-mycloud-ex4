// Package platform abstracts the NAS hardware so the rest of MoodNAS runs
// identically on the real WD My Cloud EX4 and in the simulator.
package platform

import (
	"errors"
	"time"
)

var ErrNotSupported = errors.New("not supported on this platform yet")

type Sensors struct {
	CPUTempC   float64 `json:"cpuTempC"`
	BoardTempC float64 `json:"boardTempC"`
	FanPct     int     `json:"fanPct"`
	FanRPM     int     `json:"fanRpm"`
}

type SMART struct {
	Healthy        bool    `json:"healthy"`
	TempC          float64 `json:"tempC"`
	PowerOnHours   int     `json:"powerOnHours"`
	Reallocated    int     `json:"reallocated"`    // attr 5
	ReportedUncorr int     `json:"reportedUncorr"` // attr 187
	CommandTimeout int     `json:"commandTimeout"` // attr 188
	Pending        int     `json:"pending"`        // attr 197
	OfflineUncorr  int     `json:"offlineUncorr"`  // attr 198
	CRCErrors      int     `json:"crcErrors"`      // attr 199
}

type Disk struct {
	Bay       int    `json:"bay"` // 1..4, 0 = USB/other
	Device    string `json:"device"`
	Model     string `json:"model"`
	Serial    string `json:"serial"`
	SizeBytes uint64 `json:"sizeBytes"`
	Present   bool   `json:"present"`
	Sleeping  bool   `json:"sleeping"`
	SMART     *SMART `json:"smart,omitempty"`
}

type Array struct {
	Name      string   `json:"name"`
	Level     string   `json:"level"`
	State     string   `json:"state"` // clean | degraded | resyncing | recovering | inactive
	Members   []string `json:"members"`
	Failed    []string `json:"failed"`
	Progress  float64  `json:"progress"` // 0..100 when (re)syncing
	SizeBytes uint64   `json:"sizeBytes"`
	UsedBytes uint64   `json:"usedBytes"`
	Mount     string   `json:"mount"`
}

type NetIf struct {
	Name    string `json:"name"`
	Up      bool   `json:"up"`
	Speed   int    `json:"speedMbps"`
	Address string `json:"address"`
	RxBps   uint64 `json:"rxBps"`
	TxBps   uint64 `json:"txBps"`
}

type System struct {
	Model    string        `json:"model"`
	Kernel   string        `json:"kernel"`
	Uptime   time.Duration `json:"uptime"`
	Load1    float64       `json:"load1"`
	MemTotal uint64        `json:"memTotal"`
	MemUsed  uint64        `json:"memUsed"`
	CPUPct   float64       `json:"cpuPct"`
	Net      []NetIf       `json:"net"`
}

// FileEvent feeds the ransomware sentinel (fanotify on Linux, synthetic in
// the simulator).
type FileEvent struct {
	Share   string
	Kind    string // write | rename | delete
	Path    string
	Entropy float64 // bits per byte of the written block, 0..8
	At      time.Time
}

type ButtonEvent struct {
	Name string // power | reset | func | usbcopy
	At   time.Time
}

// Panel is what MoodNAS asks the front OLED to show.
type Panel struct {
	Mood  string `json:"mood"`
	Line1 string `json:"line1"`
	Line2 string `json:"line2"`
}

type Platform interface {
	Name() string
	Sensors() (Sensors, error)
	SetFan(pct int) error
	ShowPanel(p Panel) error
	LastPanel() Panel
	Disks() ([]Disk, error)
	Arrays() ([]Array, error)
	System() (System, error)
	FileEvents() <-chan FileEvent
	Buttons() <-chan ButtonEvent
	// SetReadOnly freezes a share (sentinel response).
	SetReadOnly(share string, ro bool) error
}
