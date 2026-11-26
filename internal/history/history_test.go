package history

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQueryAveragesAndGaps(t *testing.T) {
	r := New("", time.Minute)
	t0 := time.Unix(1_800_000_000, 0)
	// 30 minutes of data, then a 30-minute gap
	for i := 0; i < 30; i++ {
		r.Add(Sample{T: t0.Add(time.Duration(i) * time.Minute).Unix(), CPU: float64(i), Load: 0.5, Mem: 40, CPUTemp: 50, DiskTemp: Unknown})
	}
	pts := r.Query(t0, t0.Add(time.Hour), 6) // 10-minute buckets
	if len(pts) != 6 {
		t.Fatalf("want 6 points, got %d", len(pts))
	}
	if pts[0].CPU == nil || *pts[0].CPU != 4.5 || *pts[0].Load != 0.5 || *pts[0].CPUTemp != 50 {
		t.Fatalf("first bucket %+v", pts[0])
	}
	if pts[0].DiskTemp != nil {
		t.Fatal("unknown disk temperature must be null, not -1")
	}
	if pts[4].CPU != nil || pts[5].CPU != nil {
		t.Fatal("gap buckets must be null")
	}
}

func TestRetentionAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.json")
	r := New(path, time.Hour) // max 360 samples
	now := time.Now()
	for i := 0; i < 400; i++ {
		r.Add(Sample{T: now.Add(time.Duration(i-400) * time.Minute).Unix(), CPU: 1})
	}
	if len(r.samples) != 360 {
		t.Fatalf("cap not applied: %d", len(r.samples))
	}
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	r2 := New(path, time.Hour)
	if len(r2.samples) != 360 {
		t.Fatalf("reload: %d samples", len(r2.samples))
	}
}
