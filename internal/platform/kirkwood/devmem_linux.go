//go:build linux

package kirkwood

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// Sensor maps the single register page read-only. It is only opened on a
// Feroceon (Kirkwood) CPU, so other hardware is never probed.
type Sensor struct {
	page []byte
	off  int
}

var (
	once   sync.Once
	sensor *Sensor
)

// Open returns the shared sensor, or nil when unavailable (not Kirkwood,
// no /dev/mem, or not running as root).
func Open() *Sensor {
	once.Do(func() {
		info, err := os.ReadFile("/proc/cpuinfo")
		if err != nil || !strings.Contains(string(info), "Feroceon") {
			return
		}
		f, err := os.OpenFile("/dev/mem", os.O_RDONLY|syscall.O_SYNC, 0)
		if err != nil {
			return
		}
		defer f.Close()
		// 64-bit arithmetic: the physical address does not fit a 32-bit int
		pageSize := int64(os.Getpagesize())
		phys := int64(RegPhys)
		base := phys &^ (pageSize - 1)
		page, err := syscall.Mmap(int(f.Fd()), base, int(pageSize), syscall.PROT_READ, syscall.MAP_SHARED)
		if err != nil {
			return
		}
		s := &Sensor{page: page, off: int(phys - base)}
		if _, ok := s.Read(); !ok {
			syscall.Munmap(page)
			return
		}
		sensor = s
	})
	return sensor
}

// Read returns the die temperature in °C.
func (s *Sensor) Read() (float64, bool) {
	if s == nil {
		return 0, false
	}
	// atomic load = one real 32-bit read of the device register
	reg := atomic.LoadUint32((*uint32)(unsafe.Pointer(&s.page[s.off])))
	return TempFromReg(reg)
}
