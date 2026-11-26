package kirkwood

import "testing"

func TestTempFromReg(t *testing.T) {
	reg := func(v uint32) uint32 { return v<<tempShift | 1<<validBit }
	// (3153000000 - 10000000*250) / 13825 = 47233 m°C
	if c, ok := TempFromReg(reg(250)); !ok || c != 47.2 {
		t.Fatalf("250 -> %v %v", c, ok)
	}
	if c, ok := TempFromReg(reg(200)); !ok || c != 83.4 {
		t.Fatalf("200 -> %v %v", c, ok)
	}
	if _, ok := TempFromReg(250 << tempShift); ok {
		t.Fatal("reading without the valid bit accepted")
	}
	if _, ok := TempFromReg(reg(0)); ok {
		t.Fatal("implausible 228 °C accepted")
	}
}
