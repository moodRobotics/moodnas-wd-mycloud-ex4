// Package kirkwood reads the Marvell Kirkwood (88F6281/6282) on-die
// temperature sensor. WD's 3.2 kernel has no driver exposing it, so when
// /sys/class/thermal is empty we read the sensor's status register directly,
// read-only, the same way mainline's kirkwood_thermal driver does.
package kirkwood

// Register: SoC internal registers base 0xF1000000 + 0x10078 (Thermal
// Manager status). Layout from drivers/thermal/kirkwood_thermal.c.
const (
	RegPhys     = 0xF1010078
	validBit    = 9
	tempShift   = 10
	tempMask    = 0x1FF
	formulaBase = 3153000000
	formulaStep = 10000000
	formulaDiv  = 13825
)

// TempFromReg converts the raw register to °C; ok=false if the sensor flags
// the reading invalid or the result is implausible.
func TempFromReg(reg uint32) (float64, bool) {
	if (reg>>validBit)&1 == 0 {
		return 0, false
	}
	v := int64((reg >> tempShift) & tempMask)
	milli := (formulaBase - formulaStep*v) / formulaDiv
	c := float64(milli) / 1000
	if c < -20 || c > 150 {
		return 0, false
	}
	return float64(int(c*10+0.5)) / 10, true
}
