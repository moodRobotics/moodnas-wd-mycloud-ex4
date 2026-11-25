// Package mcu speaks the WD My Cloud EX4 front-panel microcontroller
// protocol, as documented in docs/hardware/mcu-protocol.md (clean-room,
// reconstructed from disassembly; to be confirmed on hardware).
//
// Frames: FA <class> <cmd> <sub> <len> <data...> FB, no checksum.
package mcu

import "errors"

const (
	Start = 0xFA
	End   = 0xFB
)

var ErrNoAck = errors.New("mcu: no ACK after retries")

// Frame builds a host->MCU frame.
func Frame(class, cmd, sub, length byte, data ...byte) []byte {
	if len(data) == 0 {
		data = []byte{0}
	}
	f := []byte{Start, class, cmd, sub, length}
	f = append(f, data...)
	return append(f, End)
}

// FanControl sets the fan level 0 (stop) .. 4 (max).
func FanControl(level int) []byte {
	return Frame(0x02, 0x00, byte(clamp(level, 0, 4)), 0x00)
}

// GetThermistor asks for 1 = SATA board, 2 = main board.
func GetThermistor(sel byte) []byte { return Frame(0x03, 0x08, sel, 0x00) }

// FanLevelFor maps a 0..100 % request to a level. MoodNAS never stops the
// fan entirely (level 0), matching the vendor's own policy.
func FanLevelFor(pct int) int {
	switch {
	case pct <= 25:
		return 1
	case pct <= 50:
		return 2
	case pct <= 75:
		return 3
	}
	return 4
}

// PctForLevel is the inverse used for reporting.
func PctForLevel(level int) int { return []int{0, 25, 50, 75, 100}[clamp(level, 0, 4)] }

// thermistor ADC byte -> °C (NTC: higher raw = cooler). 127 = fault/≥127.
var thermTable = [192]int8{
	127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 127, 125, 122, 119,
	116, 113, 110, 108, 106, 103, 102, 100, 98, 96, 94, 93, 91, 90, 88, 87,
	85, 84, 83, 82, 81, 80, 79, 77, 76, 75, 74, 73, 72, 71, 70, 70,
	69, 68, 67, 66, 65, 65, 64, 63, 62, 62, 61, 60, 59, 59, 58, 57,
	57, 56, 55, 55, 54, 53, 53, 52, 52, 51, 51, 50, 49, 49, 48, 48,
	47, 47, 46, 46, 45, 44, 44, 43, 43, 42, 42, 41, 41, 41, 40, 40,
	39, 39, 38, 38, 37, 37, 36, 36, 35, 35, 34, 34, 33, 33, 32, 32,
	32, 31, 31, 30, 30, 30, 29, 29, 28, 28, 27, 27, 27, 26, 26, 26,
	25, 25, 24, 24, 23, 23, 23, 22, 22, 21, 21, 20, 20, 20, 19, 19,
	19, 18, 18, 17, 17, 16, 16, 16, 15, 15, 14, 14, 13, 13, 13, 13,
	12, 12, 11, 11, 10, 10, 10, 9, 9, 8, 8, 7, 7, 7, 6, 6,
	5, 5, 4, 4, 3, 3, 3, 2, 2, 1, 1, 0, 0, 0, 0, 0,
}

// TempFromRaw converts a thermistor byte; ok=false means invalid/fault.
func TempFromRaw(raw byte) (float64, bool) {
	if int(raw) >= len(thermTable) || thermTable[raw] == 127 {
		return 0, false
	}
	return float64(thermTable[raw]), true
}

// Parser splits the MCU byte stream into frames. Valid MCU frames are 7, 13
// (date/time block) or 15 bytes (version) long, start with FA and end in FB.
type Parser struct{ buf []byte }

func (p *Parser) Feed(b []byte) [][]byte {
	p.buf = append(p.buf, b...)
	var out [][]byte
	for {
		i := indexByte(p.buf, Start)
		if i < 0 {
			p.buf = p.buf[:0]
			return out
		}
		p.buf = p.buf[i:]
		matched := false
		for _, n := range []int{7, 13, 15} {
			if len(p.buf) >= n && p.buf[n-1] == End {
				out = append(out, append([]byte(nil), p.buf[:n]...))
				p.buf = p.buf[n:]
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		if len(p.buf) < 15 {
			return out // wait for more bytes
		}
		p.buf = p.buf[1:] // garbage: resync on next FA
	}
}

func IsAck(f []byte) bool { return len(f) == 7 && f[1] == 0x30 }

// ThermistorReply decodes FA 03 08 <sel> ?? <raw> FB.
func ThermistorReply(f []byte) (sel byte, tempC float64, ok bool) {
	if len(f) != 7 || f[1] != 0x03 || f[2] != 0x08 {
		return 0, 0, false
	}
	t, ok := TempFromRaw(f[5])
	return f[3], t, ok
}

func indexByte(b []byte, c byte) int {
	for i, v := range b {
		if v == c {
			return i
		}
	}
	return -1
}

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }
