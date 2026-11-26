//go:build linux

package linux

import "testing"

// Paths from a real EX4 (Stage 0 report) plus mainline-style and USB paths.
func TestBayFromPath(t *testing.T) {
	cases := map[string]int{
		"../devices/pci0000:00/0000:00:01.0/host0/target0:0:0/0:0:0:0/block/sda": 1,
		"../devices/pci0000:00/0000:00:01.0/host3/target3:0:0/3:0:0:0/block/sdd": 4,
		"../devices/platform/soc/ata2/host1/target1:0:0/1:0:0:0/block/sdb":       2,
		"../devices/pci0000:01/0000:01:01.0/usb1/1-1/1-1:1.0/host4/block/sde":    0,
	}
	for p, want := range cases {
		if got := bayFromPath(p); got != want {
			t.Errorf("%s: bay %d, want %d", p, got, want)
		}
	}
}
