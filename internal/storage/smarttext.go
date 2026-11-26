package storage

import (
	"bufio"
	"errors"
	"strconv"
	"strings"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

// ParseSmartctlText parses the classic text output of `smartctl -i -H -A`.
// WD's firmware ships smartctl 5.38 (2008), which predates --json.
func ParseSmartctlText(out string) (platform.Disk, error) {
	d := platform.Disk{Present: true}
	s := &platform.SMART{Healthy: true}
	inTable := false
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Device Model:"):
			d.Model = strings.TrimSpace(strings.TrimPrefix(line, "Device Model:"))
		case strings.HasPrefix(line, "Serial Number:"):
			d.Serial = strings.TrimSpace(strings.TrimPrefix(line, "Serial Number:"))
		case strings.HasPrefix(line, "User Capacity:"):
			v := strings.TrimSpace(strings.TrimPrefix(line, "User Capacity:"))
			v = strings.ReplaceAll(strings.ReplaceAll(strings.Fields(v + " x")[0], ",", ""), ".", "")
			d.SizeBytes, _ = strconv.ParseUint(v, 10, 64)
		case strings.Contains(line, "overall-health self-assessment test result:"):
			s.Healthy = strings.Contains(line, "PASSED")
		case strings.HasPrefix(line, "ID#"):
			inTable = true
		case inTable:
			f := strings.Fields(line)
			if len(f) < 10 {
				inTable = len(f) != 0
				continue
			}
			id, err := strconv.Atoi(f[0])
			if err != nil {
				continue
			}
			raw, _ := strconv.ParseInt(f[9], 10, 64) // "39 (Min/Max 20/45)" -> 39
			if f[8] != "-" {                         // WHEN_FAILED set: attribute below threshold
				s.Healthy = false
			}
			v := int(raw)
			switch id {
			case 5:
				s.Reallocated = v
			case 9:
				s.PowerOnHours = v
			case 187:
				s.ReportedUncorr = v
			case 188:
				s.CommandTimeout = int(raw & 0xFFFF)
			case 194:
				s.TempC = float64(raw & 0xFF)
			case 197:
				s.Pending = v
			case 198:
				s.OfflineUncorr = v
			case 199:
				s.CRCErrors = v
			}
		}
	}
	if d.Model == "" {
		return d, errors.New("not smartctl output")
	}
	d.SMART = s
	return d, nil
}
