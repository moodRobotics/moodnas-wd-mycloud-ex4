package storage

import (
	"strings"
	"testing"
)

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

// Real output from a WD My Cloud EX4 (smartctl 5.38) whose bay-3 disk was
// degrading while the RAID rebuilt. Serial anonymised.
const smart538 = `smartctl version 5.38 [arm-mv5sft-linux-gnueabi] Copyright (C) 2002-8 Bruce Allen

=== START OF INFORMATION SECTION ===
Device Model:     WDC WD20EFRX-68EUZN0
Serial Number:    WD-WMC4M0000000
Firmware Version: 80.00A80
User Capacity:    2,000,398,934,016 bytes

=== START OF READ SMART DATA SECTION ===
SMART Attributes Data Structure revision number: 16
Vendor Specific SMART Attributes with Thresholds:
ID# ATTRIBUTE_NAME          FLAG     VALUE WORST THRESH TYPE      UPDATED  WHEN_FAILED RAW_VALUE
  1 Raw_Read_Error_Rate     0x002f   200   200   051    Pre-fail  Always       -       14
  5 Reallocated_Sector_Ct   0x0033   196   196   140    Pre-fail  Always       -       125
  7 Seek_Error_Rate         0x002e   197   196   000    Old_age   Always       -       348
  9 Power_On_Hours          0x0032   019   019   000    Old_age   Always       -       59796
194 Temperature_Celsius     0x0022   105   093   000    Old_age   Always       -       42
196 Reallocated_Event_Count 0x0032   197   197   000    Old_age   Always       -       3
197 Current_Pending_Sector  0x0032   200   200   000    Old_age   Always       -       0
198 Offline_Uncorrectable   0x0030   100   253   000    Old_age   Offline      -       0
199 UDMA_CRC_Error_Count    0x0032   200   200   000    Old_age   Always       -       0
`

func TestParseSmartctlText(t *testing.T) {
	d, err := ParseSmartctlText(smart538)
	if err != nil {
		t.Fatal(err)
	}
	s := d.SMART
	if d.Model != "WDC WD20EFRX-68EUZN0" || d.SizeBytes != 2000398934016 {
		t.Fatalf("identity wrong: %+v", d)
	}
	if s.Reallocated != 125 || s.PowerOnHours != 59796 || s.TempC != 42 || !s.Healthy {
		t.Fatalf("attributes wrong: %+v", s)
	}
	if _, err := ParseSmartctlText("smartctl: unrecognized option '--json'"); err == nil {
		t.Fatal("usage text accepted as SMART data")
	}
	failed := strings.Replace(smart538, "Always       -       125", "Always   FAILING_NOW 125", 1)
	if d, _ := ParseSmartctlText(failed); d.SMART.Healthy {
		t.Fatal("WHEN_FAILED not detected")
	}
}

// Real /proc/mdstat from the same unit: RAID10 rebuilding onto bay 3.
func TestParseMdstatRaid10Recovery(t *testing.T) {
	a := ParseMdstat(`Personalities : [linear] [raid0] [raid1] [raid10] [raid6] [raid5] [raid4] 
md1 : active raid10 sdc2[2] sda2[0] sdd2[3] sdb2[1]
      3898637952 blocks super 1.0 64K chunks 2 near-copies [4/3] [UU_U]
      [=>...................]  recovery =  9.1% (178250752/1949318976) finish=2463.1min speed=11983K/sec
      
md0 : active raid1 sdc1[2] sdd1[3] sdb1[1] sda1[0]
      2097088 blocks [4/4] [UUUU]
      bitmap: 0/16 pages [0KB], 8KB chunk

unused devices: <none>
`)
	if len(a) != 2 || a[0].Level != "raid10" || a[0].State != "recovering" || a[0].Progress != 9.1 {
		t.Fatalf("md1 wrong: %+v", a)
	}
	if a[1].Level != "raid1" || a[1].State != "clean" {
		t.Fatalf("md0 wrong: %+v", a[1])
	}
}
