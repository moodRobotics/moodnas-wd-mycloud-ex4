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
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
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

	smartJSON  int // 0 unknown, 1 smartctl has --json, -1 text only (smartctl < 7)
	smartCache map[string]cachedSMART
	cpuPrev    [2]uint64 // busy, total jiffies
}

type cachedSMART struct {
	disk platform.Disk
	at   time.Time
}

// SMART is read at most every smartEvery: querying it too often keeps
// disks from spinning down.
const smartEvery = 10 * time.Minute

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
	s := platform.Sensors{CPUTempC: -1} // -1 = unknown (WD's 3.2 kernel has no thermal zone)
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
		if sd, ok := l.smart(name); ok {
			sd.Device, sd.Bay = name, d.Bay
			if sd.SizeBytes == 0 {
				sd.SizeBytes = d.SizeBytes
			}
			d = sd
		}
		out = append(out, d)
	}
	return out, nil
}

// bayFromPath maps a /sys/block link to a front bay. On the EX4 all four
// bays hang off the PCIe Marvell 88SX7042 (sata_mv): SCSI hosts 0-3 are bays
// 1-4 (Stage 0 report: ata1..4 -> sda..sdd -> host0..3). Mainline kernels
// show ataN in the path instead.
func bayFromPath(link string) int {
	if strings.Contains(link, "/usb") {
		return 0
	}
	for i := 0; i < 4; i++ {
		if strings.Contains(link, fmt.Sprintf("/host%d/", i)) || strings.Contains(link, fmt.Sprintf("/ata%d/", i+1)) {
			return i + 1
		}
	}
	return 0
}

// smart returns SMART data for a disk, cached for smartEvery. It uses
// `smartctl --json` when available and falls back to the text output of old
// versions such as the 5.38 shipped with WD's firmware.
func (l *Linux) smart(name string) (platform.Disk, bool) {
	l.mu.Lock()
	if l.smartCache == nil {
		l.smartCache = map[string]cachedSMART{}
	}
	c, cached := l.smartCache[name]
	mode := l.smartJSON
	l.mu.Unlock()
	if cached && time.Since(c.at) < smartEvery {
		return c.disk, c.disk.SMART != nil
	}
	run := func(args ...string) []byte {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		// -n standby: never wake a sleeping disk just to read SMART.
		out, _ := exec.CommandContext(ctx, "smartctl", append(args, "-n", "standby", "/dev/"+name)...).Output()
		return out
	}
	var d platform.Disk
	var err error = errNoSMART
	if mode >= 0 {
		out := run("--json", "-a")
		if d, err = storage.ParseSmartctl(out); err == nil && d.Model != "" {
			mode = 1
		} else {
			err = errNoSMART
			if mode == 0 && !strings.HasPrefix(strings.TrimSpace(string(out)), "{") {
				mode = -1 // no JSON support: remember it
			}
		}
	}
	if mode < 0 {
		d, err = storage.ParseSmartctlText(string(run("-i", "-H", "-A")))
	}
	if err != nil { // sleeping (smartctl exits early) or unreadable: keep the last data
		d = c.disk
		d.Sleeping = true
	}
	l.mu.Lock()
	l.smartJSON = mode
	l.smartCache[name] = cachedSMART{disk: d, at: time.Now()}
	l.mu.Unlock()
	return d, d.SMART != nil
}

var errNoSMART = errors.New("no SMART data")

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
		m := map[string]uint64{}
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				v, _ := strconv.ParseUint(f[1], 10, 64)
				m[strings.TrimSuffix(f[0], ":")] = v << 10
			}
		}
		avail, ok := m["MemAvailable"]
		if !ok { // kernels before 3.14, such as WD's 3.2
			avail = m["MemFree"] + m["Buffers"] + m["Cached"]
		}
		s.MemTotal, s.MemUsed = m["MemTotal"], m["MemTotal"]-min(avail, m["MemTotal"])
	}
	s.CPUPct = l.cpuPct()
	ifs, _ := os.ReadDir("/sys/class/net")
	for _, e := range ifs {
		n := e.Name()
		if n == "lo" {
			continue
		}
		state, _ := os.ReadFile("/sys/class/net/" + n + "/operstate")
		speed, _ := readInt("/sys/class/net/" + n + "/speed")
		ni := platform.NetIf{Name: n, Up: strings.TrimSpace(string(state)) == "up", Speed: max(speed, 0)}
		if ifc, err := net.InterfaceByName(n); err == nil {
			addrs, _ := ifc.Addrs()
			for _, a := range addrs {
				if ipn, ok := a.(*net.IPNet); ok && ipn.IP.To4() != nil {
					ni.Address = ipn.IP.String()
					break
				}
			}
		}
		s.Net = append(s.Net, ni)
	}
	return s, nil
}

// cpuPct is the CPU usage since the previous call, from /proc/stat.
func (l *Linux) cpuPct() float64 {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	f := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
	var total, idle uint64
	for i, v := range f[1:] {
		n, _ := strconv.ParseUint(v, 10, 64)
		total += n
		if i == 3 || i == 4 { // idle, iowait
			idle += n
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	prev := l.cpuPrev
	l.cpuPrev = [2]uint64{total - idle, total}
	if prev[1] == 0 || total <= prev[1] {
		return 0
	}
	return math.Round(float64((total-idle)-prev[0])*1000/float64(total-prev[1])) / 10
}

func (l *Linux) FileEvents() <-chan platform.FileEvent { return l.files }
func (l *Linux) Buttons() <-chan platform.ButtonEvent  { return l.buttons }

func (l *Linux) SetReadOnly(share string, ro bool) error {
	// Implemented by rewriting smb.conf and reloading Samba (phase 5).
	l.log.Warn("SetReadOnly requested", "share", share, "readOnly", ro)
	return platform.ErrNotSupported
}
