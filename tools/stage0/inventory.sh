#!/bin/sh
# MoodNAS Stage 0 — read-only hardware inventory of a WD My Cloud EX4.
#
# Run it on the ORIGINAL WD firmware over SSH (enable SSH in the WD web UI:
# Settings > Network > SSH). It changes nothing on the NAS: it only reads,
# and writes a report plus a backup of the NAND flash to a share you choose.
#
#   sh inventory.sh /shares/Public
#
# Then send us (or attach to a GitHub issue) the report-*.txt file — NOT the
# mtd backups, which contain your NAS's MAC addresses and serial number. Keep
# the backups safe: with a serial console they let you restore the NAS
# exactly as it left the factory.
#
# Busybox sh compatible. Every command is read-only; any that is missing is
# simply skipped.

set -u
OUT_DIR=${1:-/shares/Public}
STAMP=$(date +%Y%m%d-%H%M%S 2>/dev/null || echo now)
DEST="$OUT_DIR/moodnas-stage0-$STAMP"
REPORT="$DEST/report-$STAMP.txt"

if [ ! -d "$OUT_DIR" ]; then
	echo "Output folder $OUT_DIR does not exist. Pass a share path, e.g. /shares/Public" >&2
	exit 1
fi
mkdir -p "$DEST/mtd" || exit 1

section() { printf '\n===== %s =====\n' "$1" >>"$REPORT"; }
run() {
	section "$*"
	sh -c "$*" >>"$REPORT" 2>&1 || echo "(exit $?)" >>"$REPORT"
}

echo "MoodNAS Stage 0 inventory — $(date)" >"$REPORT"

# --- identity and firmware
run 'cat /etc/version /usr/local/modules/files/VERSION 2>/dev/null; cat /etc/model 2>/dev/null'
run 'uname -a'
run 'cat /proc/cpuinfo'
run 'cat /proc/meminfo | head -5'
run 'cat /proc/cmdline'
run 'cat /proc/mtd'
run 'cat /proc/partitions'
run 'cat /proc/mounts'
run 'cat /proc/mdstat'
run 'dmesg'

# --- kexec support (decides how MoodNAS can boot without touching the flash)
run 'ls -l /proc/config.gz; zcat /proc/config.gz 2>/dev/null | grep -E "KEXEC|ATAGS|DTB|CMDLINE"'
run 'grep -i kexec /proc/kallsyms | head -5'
run 'which kexec; ls /sbin/kexec /usr/sbin/kexec 2>/dev/null'

# --- buses and devices
run 'cat /proc/bus/pci/devices; lspci -nn 2>/dev/null'
run 'ls -l /sys/block/; for d in /sys/block/sd*; do echo "$d -> $(readlink -f $d)"; done'
run 'cat /proc/bus/input/devices 2>/dev/null'
run 'ls -l /dev/i2c* /dev/ttyS* 2>/dev/null'
run 'i2cdetect -y 0 2>/dev/null; i2cdetect -y 1 2>/dev/null'
run 'cat /sys/kernel/debug/gpio 2>/dev/null'
run 'ls /sys/class/hwmon/ /sys/class/thermal/ 2>/dev/null; cat /sys/class/hwmon/*/name /sys/class/hwmon/*/temp*_input 2>/dev/null'
run 'ip addr 2>/dev/null || ifconfig -a'
run 'ethtool egiga0 2>/dev/null; ethtool egiga1 2>/dev/null'

# --- disks (SMART read is passive)
for d in /dev/sda /dev/sdb /dev/sdc /dev/sdd; do
	[ -b "$d" ] && run "smartctl -i -A $d"
done

# --- U-Boot environment (read-only)
run 'fw_printenv 2>/dev/null || /usr/local/sbin/fw_printenv 2>/dev/null'
run 'dd if=/dev/mtd0 bs=1024 count=1024 2>/dev/null | strings | grep -E "^(bootcmd|bootargs|mtdparts|ethaddr|eth1addr|bootdelay|baudrate|console|U-Boot)" | sort -u'

# --- MCU protocol confirmation: passive observation only
if command -v strace >/dev/null 2>&1; then
	pid=$(pidof up_send_daemon 2>/dev/null)
	if [ -n "$pid" ]; then
		section "strace up_send_daemon while querying the MCU (read-only queries)"
		strace -f -tt -e trace=write,read -xx -p "$pid" -o "$DEST/strace-send.txt" &
		sp=$!
		sleep 1
		up_send_ctl GetThemister 2 >/dev/null 2>&1
		up_send_ctl MCUVer >/dev/null 2>&1
		up_send_ctl RDateAndTime >/dev/null 2>&1
		sleep 2
		kill "$sp" 2>/dev/null
		head -100 "$DEST/strace-send.txt" >>"$REPORT"
	fi
else
	section "strace not available: MCU protocol not traced"
fi

# --- NAND backup (reading /dev/mtdX is safe; nothing is written to flash)
section "NAND backup"
for m in /dev/mtd[0-9]; do
	[ -c "$m" ] || continue
	name=$(basename "$m")
	if dd if="$m" of="$DEST/mtd/$name.img" bs=64k 2>/dev/null; then
		echo "$name: $(wc -c <"$DEST/mtd/$name.img") bytes, sha256 $(sha256sum "$DEST/mtd/$name.img" | cut -d' ' -f1)" >>"$REPORT"
	fi
done

echo
echo "Done. Report: $REPORT"
echo "NAND backup: $DEST/mtd/  (keep it private: it contains your MAC addresses and serial number)"
