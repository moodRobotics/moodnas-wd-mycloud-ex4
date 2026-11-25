//go:build linux

// Package linux is the real-hardware platform for the WD My Cloud EX4
// running a mainline kernel. Storage, sensors and system data come from
// /proc, /sys and smartctl. Fan, OLED and buttons go through the front-panel
// microcontroller on /dev/ttyS1; that driver is filled in once the protocol in
// docs/hardware/mcu-protocol.md is confirmed on hardware. Until then those
// calls return platform.ErrNotSupported and the MCU keeps its own built-in
// fan policy, which is the safe default.
package linux

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform/mcu"
	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/storage"
)

type Linux struct {
	log        *slog.Logger
	mu         sync.Mutex
	panel      platform.Panel
	files      chan platform.FileEvent
	buttons    chan platform.ButtonEvent
	sharesRoot string
	mcu        *mcu.Client
	fanLevel   int
}

// AttachMCU enables the experimental front-panel MCU driver.
func (l *Linux) AttachMCU(c *mcu.Client) {
	l.mcu = c
	l.fanLevel = -1
}

func New(log *slog.Logger, sharesRoot string) *Linux {
	return &Linux{log: log, files: make(chan platform.FileEvent, 256),
		buttons: make(chan platform.ButtonEvent, 8), sharesRoot: sharesRoot}
}

func (l *Linux) Name() string { return "wd-mycloud-ex4" }

func readInt(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(b)))
}

func (l *Linux) Sensors() (platform.Sensors, error) {
	var s platform.Sensors
	// Kirkwood die sensor (kirkwood-thermal) and any hwmon (e.g. lm75 at 0x4c).
	if v, err := readInt("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		s.CPUTempC = float64(v) / 1000
	}
	hw, _ := filepath.Glob("/sys/class/hwmon/hwmon*/temp1_input")
	for _, p := range hw {
		name, _ := os.ReadFile(filepath.Join(filepath.Dir(p), "name"))
		if strings.HasPrefix(strings.TrimSpace(string(name)), "lm75") {
			if v, err := readInt(p); err == nil {
				s.BoardTempC = float64(v) / 1000
			}
		}
	}
	s.FanPct = -1 // unknown without the MCU driver
	if l.mcu != nil {
		if t, ok := l.mcu.Thermistor(2); ok {
			s.BoardTempC = t
		}
		l.mu.Lock()
		if l.fanLevel >= 0 {
			s.FanPct = mcu.PctForLevel(l.fanLevel)
		}
		l.mu.Unlock()
	}
	return s, nil
}

func (l *Linux) SetFan(pct int) error {
	if l.mcu == nil {
		return platform.ErrNotSupported
	}
	level := mcu.FanLevelFor(pct)
	l.mu.Lock()
	same := level == l.fanLevel
	l.mu.Unlock()
	if same {
		return nil
	}
	if err := l.mcu.Send(mcu.FanControl(level)); err != nil {
		return err
	}
	l.mu.Lock()
	l.fanLevel = level
	l.mu.Unlock()
	return nil
}

func (l *Linux) ShowPanel(p platform.Panel) error {
	l.mu.Lock()
	l.panel = p
	l.mu.Unlock()
	return platform.ErrNotSupported
}

func (l *Linux) LastPanel() platform.Panel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.panel
}

func (l *Linux) Disks() ([]platform.Disk, error) {
	ents, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil, err
	}
	var out []platform.Disk
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "sd") {
			continue
		}
		d := platform.Disk{Device: name, Present: true}
		if n, err := readInt("/sys/block/" + name + "/size"); err == nil {
			d.SizeBytes = uint64(n) * 512
		}
		link, _ := os.Readlink("/sys/block/" + name)
		d.Bay = bayFromPath(link)
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		// -n standby: never wake a sleeping disk just to read SMART.
		b, err := exec.CommandContext(ctx, "smartctl", "--json", "-a", "-n", "standby", "/dev/"+name).Output()
		cancel()
		if len(b) > 0 {
			if sd, perr := storage.ParseSmartctl(b); perr == nil && sd.SMART != nil && sd.Model != "" {
				sd.Device, sd.Bay = name, d.Bay
				if sd.SizeBytes == 0 {
					sd.SizeBytes = d.SizeBytes
				}
				d = sd
			} else if err != nil {
				d.Sleeping = true
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// bayFromPath maps a /sys/block link to a front bay. The exact mapping of
// ata ports to bays is confirmed by tools/stage0 on real hardware; until
// then we fall back to device order.
func bayFromPath(link string) int {
	for i, key := range []string{"ata1/", "ata2/", "ata3/", "ata4/"} {
		if strings.Contains(link, key) {
			return i + 1
		}
	}
	return 0
}

func (l *Linux) Arrays() ([]platform.Array, error) {
	b, err := os.ReadFile("/proc/mdstat")
	if err != nil {
		return nil, err
	}
	arr := storage.ParseMdstat(string(b))
	mounts := readMounts()
	for i := range arr {
		if mp, ok := mounts["/dev/"+arr[i].Name]; ok {
			arr[i].Mount = mp
			if total, used, err := statfs(mp); err == nil {
				arr[i].SizeBytes, arr[i].UsedBytes = total, used
			}
		}
	}
	return arr, nil
}

func readMounts() map[string]string {
	m := map[string]string{}
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 {
			m[fields[0]] = fields[1]
		}
	}
	return m
}

func (l *Linux) System() (platform.System, error) {
	s := platform.System{Model: "WD My Cloud EX4"}
	if b, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		s.Kernel = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/proc/uptime"); err == nil {
		var up float64
		fmt.Sscanf(string(b), "%f", &up)
		s.Uptime = time.Duration(up * float64(time.Second))
	}
	if b, err := os.ReadFile("/proc/loadavg"); err == nil {
		fmt.Sscanf(string(b), "%f", &s.Load1)
	}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail uint64
		for _, line := range strings.Split(string(b), "\n") {
			var v uint64
			if n, _ := fmt.Sscanf(line, "MemTotal: %d kB", &v); n == 1 {
				total = v << 10
			}
			if n, _ := fmt.Sscanf(line, "MemAvailable: %d kB", &v); n == 1 {
				avail = v << 10
			}
		}
		s.MemTotal, s.MemUsed = total, total-avail
	}
	ifs, _ := os.ReadDir("/sys/class/net")
	for _, e := range ifs {
		n := e.Name()
		if n == "lo" {
			continue
		}
		state, _ := os.ReadFile("/sys/class/net/" + n + "/operstate")
		speed, _ := readInt("/sys/class/net/" + n + "/speed")
		s.Net = append(s.Net, platform.NetIf{Name: n, Up: strings.TrimSpace(string(state)) == "up", Speed: speed})
	}
	return s, nil
}

func (l *Linux) FileEvents() <-chan platform.FileEvent { return l.files }
func (l *Linux) Buttons() <-chan platform.ButtonEvent  { return l.buttons }

func (l *Linux) SetReadOnly(share string, ro bool) error {
	// Implemented by rewriting smb.conf and reloading Samba (phase 5).
	l.log.Warn("SetReadOnly requested", "share", share, "readOnly", ro)
	return platform.ErrNotSupported
}
