#!/bin/sh
# MoodNAS sidecar: run MoodNAS on a WD My Cloud EX4 *next to* the original
# WD firmware. Nothing is installed and nothing is written to the flash:
# everything lives in this folder, and `stop` + deleting the folder removes it.
#
#   sh moodnas-sidecar.sh start     # then open https://<nas-ip>:8443
#   sh moodnas-sidecar.sh status
#   sh moodnas-sidecar.sh log
#   sh moodnas-sidecar.sh stop
#
# In sidecar mode MoodNAS only observes: disks, SMART, RAID, network, load.
# Fan, front display and buttons stay with WD's firmware, which owns them.
# It runs at the lowest CPU priority with a 64 MiB memory target and reads
# SMART at most every 10 minutes, so it does not slow down a RAID rebuild.

DIR=$(cd "$(dirname "$0")" && pwd)
PORT=${MOODNAS_PORT:-8443}
PIDFILE="$DIR/moodd.pid"
LOG="$DIR/moodd.log"

running() {
	[ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null
}

case "$1" in
start)
	if running; then
		echo "MoodNAS is already running (pid $(cat "$PIDFILE"))."
		exit 0
	fi
	chmod +x "$DIR/moodd" 2>/dev/null
	mkdir -p "$DIR/data"
	[ -f "$LOG" ] && mv "$LOG" "$LOG.1"
	GOMEMLIMIT=64MiB nohup nice -n 19 "$DIR/moodd" \
		-listen ":$PORT" -http "" -interval 60s \
		-data "$DIR/data" -ui "$DIR/ui" >>"$LOG" 2>&1 &
	echo $! >"$PIDFILE"
	sleep 4
	if ! running; then
		echo "MoodNAS failed to start. Last log lines:"
		tail -20 "$LOG"
		exit 1
	fi
	IP=$(ip -4 addr show bond0 2>/dev/null | sed -n 's/.*inet \([0-9.]*\).*/\1/p' | head -1)
	echo "MoodNAS is running: https://${IP:-<nas-ip>}:$PORT"
	echo "The browser will warn about the self-signed certificate; its SHA-256 is in the log."
	# moodd prints a one-time setup code only while no administrator exists
	CODE=$(sed -n 's/.*setupCode=\([A-Z0-9-]*\).*/\1/p' "$LOG" | tail -1)
	if [ -n "$CODE" ]; then
		echo
		echo "  =============================================="
		echo "   First-time setup code:  $CODE"
		echo "   (not shown on the NAS display in sidecar mode)"
		echo "  =============================================="
	fi
	;;
stop)
	if running; then
		kill "$(cat "$PIDFILE")"
		echo "MoodNAS stopped."
	else
		echo "MoodNAS is not running."
	fi
	rm -f "$PIDFILE"
	;;
status)
	if running; then
		echo "running (pid $(cat "$PIDFILE")), port $PORT"
	else
		echo "stopped"
	fi
	;;
log)
	tail -40 "$LOG"
	CODE=$(sed -n "s/.*setupCode=\([A-Z0-9-]*\).*/\1/p" "$LOG" | tail -1)
	[ -n "$CODE" ] && echo && echo "Setup code: $CODE"
	;;
*)
	echo "usage: sh $0 {start|stop|status|log}"
	exit 1
	;;
esac
