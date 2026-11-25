package storage

import "testing"

const mdstat = `Personalities : [linear] [raid0] [raid1] [raid10] [raid6] [raid5] [raid4]
md1 : active raid5 sdd2[3] sdc2[2](F) sdb2[1] sda2[0]
      8778405888 blocks super 1.0 level 5, 64k chunk, algorithm 2 [4/3] [UU_U]
      [==>..................]  recovery = 12.6% (369000000/2926135296) finish=300.0min speed=140000K/sec

md0 : active raid1 sdd1[3] sdc1[2] sdb1[1] sda1[0]
      2097088 blocks [4/4] [UUUU]

md2 : inactive sde1[0](S)
      976630336 blocks

unused devices: <none>
`

func TestParseMdstat(t *testing.T) {
	a := ParseMdstat(mdstat)
	if len(a) != 3 {
		t.Fatalf("want 3 arrays, got %d: %+v", len(a), a)
	}
	if a[0].Name != "md1" || a[0].Level != "raid5" || a[0].State != "recovering" || a[0].Progress != 12.6 {
		t.Fatalf("md1 wrong: %+v", a[0])
	}
	if len(a[0].Failed) != 1 || a[0].Failed[0] != "sdc2" || len(a[0].Members) != 3 {
		t.Fatalf("md1 members wrong: %+v", a[0])
	}
	if a[0].SizeBytes != 8778405888*1024 {
		t.Fatalf("md1 size wrong: %d", a[0].SizeBytes)
	}
	if a[1].State != "clean" || a[1].Level != "raid1" {
		t.Fatalf("md0 wrong: %+v", a[1])
	}
	if a[2].State != "inactive" {
		t.Fatalf("md2 wrong: %+v", a[2])
	}
}

func TestParseSmartctl(t *testing.T) {
	j := `{"model_name":"WDC WD40EFRX","serial_number":"WD-1","user_capacity":{"bytes":4000787030016},
	"smart_status":{"passed":true},"temperature":{"current":38},"power_on_time":{"hours":41000},
	"ata_smart_attributes":{"table":[{"id":5,"raw":{"value":8}},{"id":197,"raw":{"value":2}},
	{"id":188,"raw":{"value":4295032833}}]}}`
	d, err := ParseSmartctl([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	s := d.SMART
	if d.Model != "WDC WD40EFRX" || s.Reallocated != 8 || s.Pending != 2 || s.TempC != 38 || s.PowerOnHours != 41000 {
		t.Fatalf("parse wrong: %+v %+v", d, s)
	}
	if s.CommandTimeout != 1 { // 4295032833 = 0x1_0001_0001 -> low 16 bits = 1
		t.Fatalf("cmd timeout masking wrong: %d", s.CommandTimeout)
	}
}
