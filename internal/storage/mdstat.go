// Package storage parses Linux storage state (mdadm, smartctl) into
// platform types. It is pure parsing so it is fully unit-testable.
package storage

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"

	"github.com/moodrobotics/moodnas-wd-mycloud-ex4/internal/platform"
)

var (
	reHeader   = regexp.MustCompile(`^(md\w+)\s*:\s*(\w+)\s*(?:\((?:auto-)?read-only\)\s*)?(raid\d+|linear)?\s*(.*)$`)
	reMember   = regexp.MustCompile(`(\w+)\[\d+\](\([FS]\))?`)
	reBlocks   = regexp.MustCompile(`^\s*(\d+) blocks`)
	reStatus   = regexp.MustCompile(`\[(\d+)/(\d+)\]\s*\[([U_]+)\]`)
	reProgress = regexp.MustCompile(`(resync|recovery|reshape|check)\s*=\s*([\d.]+)%`)
)

// ParseMdstat parses /proc/mdstat.
func ParseMdstat(s string) []platform.Array {
	var out []platform.Array
	var cur *platform.Array
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		line := sc.Text()
		if m := reHeader.FindStringSubmatch(line); m != nil {
			out = append(out, platform.Array{Name: m[1], State: "clean", Level: m[3]})
			cur = &out[len(out)-1]
			if m[2] == "inactive" {
				cur.State = "inactive"
			}
			for _, mm := range reMember.FindAllStringSubmatch(m[4], -1) {
				if mm[2] == "(F)" {
					cur.Failed = append(cur.Failed, mm[1])
				} else {
					cur.Members = append(cur.Members, mm[1])
				}
			}
			continue
		}
		if cur == nil {
			continue
		}
		if m := reBlocks.FindStringSubmatch(line); m != nil {
			n, _ := strconv.ParseUint(m[1], 10, 64)
			cur.SizeBytes = n * 1024
		}
		if m := reStatus.FindStringSubmatch(line); m != nil {
			if strings.Contains(m[3], "_") && cur.State == "clean" {
				cur.State = "degraded"
			}
		}
		if m := reProgress.FindStringSubmatch(line); m != nil {
			cur.Progress, _ = strconv.ParseFloat(m[2], 64)
			if m[1] == "recovery" {
				cur.State = "recovering"
			} else {
				cur.State = "resyncing"
			}
		}
		if strings.TrimSpace(line) == "" {
			cur = nil
		}
	}
	return out
}
