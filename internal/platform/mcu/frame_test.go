package mcu

import (
	"bytes"
	"testing"
)

func TestFrames(t *testing.T) {
	if got := FanControl(3); !bytes.Equal(got, []byte{0xFA, 0x02, 0x00, 0x03, 0x00, 0x00, 0xFB}) {
		t.Fatalf("FanControl(3) = % X", got)
	}
	if got := FanControl(9); got[3] != 4 {
		t.Fatalf("level not clamped: % X", got)
	}
	if got := GetThermistor(2); !bytes.Equal(got, []byte{0xFA, 0x03, 0x08, 0x02, 0x00, 0x00, 0xFB}) {
		t.Fatalf("GetThermistor(2) = % X", got)
	}
}

func TestFanLevel(t *testing.T) {
	for pct, want := range map[int]int{0: 1, 20: 1, 40: 2, 60: 3, 100: 4} {
		if got := FanLevelFor(pct); got != want {
			t.Errorf("FanLevelFor(%d)=%d want %d", pct, got, want)
		}
	}
}

func TestTemp(t *testing.T) {
	if c, ok := TempFromRaw(0x40); !ok || c != 57 {
		t.Fatalf("raw 0x40 = %v %v", c, ok)
	}
	if _, ok := TempFromRaw(0x00); ok {
		t.Fatal("127 must be treated as fault")
	}
	if _, ok := TempFromRaw(0xC5); ok {
		t.Fatal("out-of-table raw accepted")
	}
}

func TestParser(t *testing.T) {
	var p Parser
	stream := []byte{0x00, 0x11, // noise
		0xFA, 0x30, 0, 0, 0, 0, 0xFB, // ACK
		0xFA, 0x03, 0x08, 0x02, 0x00} // half a thermistor reply
	fr := p.Feed(stream)
	if len(fr) != 1 || !IsAck(fr[0]) {
		t.Fatalf("frames %v", fr)
	}
	fr = p.Feed([]byte{0x40, 0xFB})
	if len(fr) != 1 {
		t.Fatalf("second chunk frames %v", fr)
	}
	sel, c, ok := ThermistorReply(fr[0])
	if !ok || sel != 2 || c != 57 {
		t.Fatalf("thermistor reply %v %v %v", sel, c, ok)
	}
	// 13-byte date/time block
	dt := append([]byte{0xFA, 0x01, 0x00, 0x03}, make([]byte, 8)...)
	dt = append(dt, 0xFB)
	if fr := p.Feed(dt); len(fr) != 1 || len(fr[0]) != 13 {
		t.Fatalf("date block %v", fr)
	}
}
