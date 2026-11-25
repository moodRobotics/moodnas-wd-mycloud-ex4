package storage

import (
	"encoding/json"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

// smartctlJSON mirrors the subset of `smartctl --json -a` output we use.
type smartctlJSON struct {
	ModelName    string `json:"model_name"`
	SerialNumber string `json:"serial_number"`
	UserCapacity struct {
		Bytes uint64 `json:"bytes"`
	} `json:"user_capacity"`
	SmartStatus struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`
	Temperature struct {
		Current float64 `json:"current"`
	} `json:"temperature"`
	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`
	ATA struct {
		Table []struct {
			ID  int `json:"id"`
			Raw struct {
				Value int64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

// ParseSmartctl converts smartctl JSON into a Disk with SMART data.
func ParseSmartctl(b []byte) (platform.Disk, error) {
	var j smartctlJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return platform.Disk{}, err
	}
	s := &platform.SMART{
		Healthy:      j.SmartStatus.Passed,
		TempC:        j.Temperature.Current,
		PowerOnHours: j.PowerOnTime.Hours,
	}
	for _, a := range j.ATA.Table {
		v := int(a.Raw.Value & 0xFFFFFFFF) // some drives pack extra data in the high bytes
		switch a.ID {
		case 5:
			s.Reallocated = v
		case 187:
			s.ReportedUncorr = v
		case 188:
			s.CommandTimeout = int(a.Raw.Value & 0xFFFF)
		case 197:
			s.Pending = v
		case 198:
			s.OfflineUncorr = v
		case 199:
			s.CRCErrors = v
		case 194:
			if s.TempC == 0 {
				s.TempC = float64(a.Raw.Value & 0xFF)
			}
		}
	}
	return platform.Disk{
		Model: j.ModelName, Serial: j.SerialNumber, SizeBytes: j.UserCapacity.Bytes,
		Present: true, SMART: s,
	}, nil
}
