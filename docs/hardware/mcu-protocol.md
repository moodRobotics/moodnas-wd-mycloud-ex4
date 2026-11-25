# WD My Cloud EX4: front-panel MCU serial protocol

Status: reverse-engineered from WD's ARM user-space binaries (static analysis only; nothing was executed).
Purpose: a clean-room specification that MoodNAS can reimplement in Go. This document records formats,
constants and byte sequences only. It does not reproduce vendor code.

Binaries analysed (from `work/mcu/`): `librs232.so`, `up_send_ctl`, `up_send_daemon`, `up_read_daemon`,
`up_send_init_info`, `libtemperature.so`, `libhwshare.so`, `fan_control`, `led`, `rtc`, `set_pwm`,
`sata_pwr_ctl`, `mcu_upgrade`, plus the shell scripts in `work/mcu/script/`.
Tooling: `arm-none-eabi-objdump/readelf/strings` in a read-only Docker mount, plus small Python table parsers.

Confidence tags used throughout:

- **[verified in disassembly]**: read directly from instructions or data tables.
- **[from strings]**: taken from help or usage text or format strings embedded in the binaries.
- **[inferred]**: a reasoned interpretation that still needs hardware confirmation.

---

## 0. TL;DR

| Item | Value |
|---|---|
| Device | `/dev/ttyS1` (SoC UART1) [verified] |
| Line settings | **19200 baud, 8 data bits, no parity, 1 stop bit, no flow control, raw mode** [verified] |
| Frame | `0xFA` + payload + `0xFB`. **There is no checksum and no escaping.** [verified] |
| Typical host→MCU frame | 7 bytes: `FA <class> <cmd> <sub> <len> <data> FB` [verified] |
| Variable frames | `FA <class> <cmd> <sub> <len> <len bytes…> FB` (strings, date/time, IP) [verified] |
| MCU→host frames | Fixed lengths of 7, 13 or 15 bytes, each ending in `FB` [verified] |
| ACK | MCU sends `FA 30 xx xx xx xx FB`. The host retransmits the whole frame if no ACK arrives within about 300 ms, for up to 10 attempts [verified host side; MCU side inferred] |
| Fan | `FA 02 00 <level 0..4> 00 00 FB` (0 = stop … 4 = max) [verified] |
| Temperature | Request `FA 03 08 <1=SATA board, 2=main board> 00 00 FB`. Reply `FA 03 08 <sel> ?? <raw> FB`, where raw maps to °C through a 192-entry table (§6.3) [verified] |

---

## 1. Serial port settings

### 1.1 Device

`up_send_daemon`, `up_read_daemon` and `mcu_upgrade` all open the literal path `/dev/ttyS1` [verified, from strings + disassembly].
The send daemon and the read daemon each open their own file descriptor on the same tty. One writes and the other reads.

### 1.2 `librs232.so` (the serial wrapper)

Exported API: `RS232_Open(path)`, `RS232_SetParam(h, *param)`, `RS232_Send(h, buf, len)`, `RS232_Read(h, buf, len)`, `RS232_Close(h)`.

- **Open** [verified]: `open(path, 0x102)`. On ARM Linux this is `O_RDWR | O_NOCTTY`. `O_NONBLOCK` is **not** set. The handle is a malloc'd struct `{int fd; struct termios t; …}`.
- **SetParam** [verified]: the param struct is `{int baud; int databits; char parity; int stopbits; int swflow; int hwflow}`. The library builds a termios from scratch:
  - `c_iflag = IGNPAR` (0x004). It adds `IXON|IXOFF` (0x1400) only if `swflow` is set, which it is not.
  - `c_oflag = 0`
  - `c_lflag = 0` (raw, non-canonical, no echo)
  - `c_cflag = CREAD | CLOCAL | Bxxxx | CSx [| PARENB] [| CSTOPB] [| CRTSCTS]`
  - Baud is mapped to the standard `B*` constants. The library supports 50…230400 (for example 19200 → `B19200` = 0x0E, 57600 → 0x1001, 115200 → 0x1002).
  - The library then calls `tcflush(fd, TCIFLUSH)` and `tcsetattr(fd, TCSANOW, &t)`.
  - **`c_cc[]` (VMIN/VTIME) is never initialised.** It comes from uninitialised `malloc` memory, so the vendor's read behaviour is effectively undefined. In practice it is probably VMIN=0/VTIME=0 (polling), because `up_read_daemon` sleeps 10 µs whenever `read()` returns 0 [inferred]. **MoodNAS should set VMIN=1, VTIME=0 explicitly, or use poll/epoll.**
- **Send** [verified]: a single `write()`. It rejects payloads larger than 512 bytes and fails on a short write.
- **Read** [verified]: a single `read()`. It returns 0 on error.

### 1.3 Parameters actually used

Both daemons pass `{19200, 8, 'N', 1, 0, 0}`. The resulting setting is **19200 8N1, no software or hardware flow control**, so `c_cflag = 0x8BE` [verified].

`mcu_upgrade` is a separate tool for the MCU's in-system-programming (ISP) bootloader. It uses other speeds: its strings mention "9600", "change to baudrate 57600" and "set br 115200" [from strings]. That bootloader protocol is **not** the run-time protocol, and it is **not** documented here.

---

## 2. Frame format

### 2.1 General structure [verified]

```
offset: 0     1        2      3      4      5 … 4+len    last
        0xFA  class    cmd    sub    len    data[len]    0xFB
```

- Start byte `0xFA` and end byte `0xFB`.
- There is **no checksum, CRC or length-complement**. The host builds frames by storing fields directly and appending `0xFB`. No code computes any sum.
- There is **no byte stuffing**. The receiver restarts the frame whenever it sees `0xFA` (see §2.3), so payloads must not contain `0xFA`. In practice payloads are small integers, BCD digits or ASCII.
- For most "simple" commands the frame is exactly 7 bytes and `len` is a nominal 0 or 1, followed by exactly one data byte (often 0). The `len` field is literally copied from a per-command table. For a few commands it is non-zero even though only one byte follows, for example `SysIPBondSet` = `FA 04 02 10 08 00 FB` [verified, quirk].
- A few frames deviate from this layout:
  - `ErrStatus` without a string is **5 bytes**: `FA 55 <idx> <en> FB`.
  - `temperature` is **10 bytes**: `FA 04 09 00 00 c_lo c_hi f_lo f_hi FB`.
  - `HDDExist` (1-argument form) is **8 bytes**: `FA 04 03 b0 b1 b2 b3 FB`.

### 2.2 Host → MCU transmission (in `up_send_daemon`) [verified]

1. The daemon receives one frame from the SysV message queue (§5).
2. It writes the frame **one byte per `write()`**, sleeping 10 µs between bytes.
3. It sleeps **300 ms**.
4. It checks whether `/runtime/mcu/ack` in WD's xmldb equals `"ack"`. `up_read_daemon` sets that key when the MCU sends its ACK frame (§2.4).
   - If the key is set, the daemon clears it (`xmldbc -d /runtime/mcu/ack`) and goes to the next message.
   - If it is not set, the daemon **retransmits the whole frame**, for up to **10 attempts** in total. After that it gives up silently.
5. Special case: a queued frame whose bytes 2..6 are `04 04 0A 00 00` is **not sent**. The daemon prints "MCU request Status" instead. This looks like a sentinel or no-op [verified; purpose unknown].

Clean-room implication: implement a single writer that sends one frame, waits up to about 300 ms for an ACK frame, and retries up to 10 times. A full-frame `write()` should be fine. The vendor's per-byte pacing is about 10 µs, which is shorter than one character time at 19200 baud (about 520 µs), so it adds nothing [inferred].

### 2.3 MCU → host reception (in `up_read_daemon`) [verified]

The receiver is a byte-at-a-time state machine over a small buffer:

- When it reads `0xFA`, it places `FA` at index 0 and restarts the frame at index 1. This happens at any position.
- After each stored byte it checks the current length:
  - **Length 7** with `buf[6] == 0xFB` → it dispatches a 7-byte frame through the response table (§4).
  - **Length 13** with `buf[12] == 0xFB` → it **always** treats the frame as the date/time block (`RDateAndTime`/`RMDateAndTime` reply), without checking command bytes.
  - **Length 15** with `buf[14] == 0xFB` → it looks up type-3 entries (only `MCUVer`).
  - Otherwise it keeps reading. There is no overflow guard beyond 15 bytes. The buffer is about 20 bytes. A robust reimplementation should resync after more than 15 bytes.
- Response matching compares `buf[1..3]` (class, cmd, sub) against a 38-entry table. Some types compare fewer bytes (§4).
- `buf[4]` appears to be a length byte (`01` for single-value replies, `07` for date/time). It is not checked [inferred].

### 2.4 ACK frame [verified host side]

MCU → host: `FA 30 00 00 xx xx FB`. Only `buf[1..3] = 30 00 00` is matched. The daemon writes `"ack"` to `/runtime/mcu/ack`.

Every frame the host sends, including queries, waits for this ACK. So the MCU almost certainly ACKs **every** command, and data replies arrive as separate unsolicited frames [inferred; confirm with strace, §9].

---

## 3. Host → MCU command table

Command names are the vendor CLI verbs (`up_send_ctl <Name> [args]`). Bytes are hexadecimal. `a1`, `a2` and so on mean the CLI's 1st, 2nd … decimal argument as a raw byte unless noted otherwise. "BCD" means the decimal value is packed as two BCD nibbles (`23` → `0x23`).

All frames below are **[verified in disassembly]** (taken from the 82-entry command table in `up_send_ctl` and its handler functions), unless they are tagged otherwise. Argument meanings marked [from strings] come from the tool's usage text.

### 3.1 Fan, temperature, power LED (class 0x02 / 0x03)

| Name | Frame | Notes |
|---|---|---|
| `FanControl n` | `FA 02 00 nn 00 00 FB` | n = 0 stop, 1 low, 2 medium, 3 high, 4 max. The host rejects n > 4. This is the **real fan PWM control**. |
| `GetThemister s` | `FA 03 08 ss 00 00 FB` | s = 1 → SATA-board thermistor, 2 → main-board thermistor. The reply is in §4. (`GetThemister 3` and `4` only read the cached xmldb value and send nothing.) |
| `PwrLedRed c` | `FA 03 10 cc 00 00 FB` | c = 0 blue off, 1 blue on, 2 blue blink, 10 (0x0A) red off, 11 (0x0B) red on, 12 (0x0C) red blink [values from strings] |
| `HddAccessLed set hdd` | `FA 03 06 ss 00 (1<<hdd) FB` | set 0/1 = disable/enable; hdd 0..n. The usage text lists hdd0–hdd2; the EX4 presumably uses 0–3 [inferred]. |

### 3.2 Power, reset, recovery, WOL (class 0x03)

| Name | Frame | Notes |
|---|---|---|
| `DeviceReady` | `FA 03 01 00 00 00 FB` | Sent at end of boot by `up_send_init_info` (the MCU probably stops its "booting" display) [inferred] |
| `APRDisable` / `APREnable` / `APRStat` | `FA 03 02 00/01/02 00 00 FB` | APR = auto power recovery after AC loss. Stat triggers an MCU report (§4, 0x41) [inferred] |
| `DeviceShutdown t` | `FA 03 03 01 01 tt FB` | Usage: "time = second". `shutdown.sh` sends `DeviceShutdown 1` and then `poweroff`, or `DeviceShutdown 0` on OLED models. The host first arms Wake-on-LAN in both Ethernet PHYs (§3.6). |
| `MCUReset t` | `FA 03 03 02 01 tt FB` | Same handler as DeviceShutdown, including the PHY WOL setup |
| `DeviceShutdownByUPS t` | `FA 03 03 03 01 tt FB` | Same handler |
| `DeviceShutdownMsg` | `FA 03 03 04 01 00 FB` | Probably only shows a shutdown message on the OLED [inferred] |
| `DeviceRstMsg` | `FA 03 03 05 01 00 FB` | Same, for reboot [inferred] |
| `MCUVer` | `FA 03 05 00 00 00 FB` | The reply is a 15-byte frame (§4) |
| `MFGMode v` | `FA 03 07 vv 00 00 FB` | 0 disable, 1 enable (factory test mode). It also writes xmldb `/runtime/MFG`. |
| `GetOnCause` | `FA 03 09 00 00 00 FB` | Asks why the box powered on. The reply is probably the `0x40` "PWR_LOST" frame [inferred] |
| `WOLDisable` / `WOLEnable` / `WOLStat` | `FA 03 12 00/01/02 00 00 FB` | Stat → MCU report `0x42` [inferred] |

### 3.3 RTC (class 0x01)

The MCU holds the battery-backed RTC. Register reads are `sub = 01` and register writes are `sub = 02`. Register numbers [verified]:

| reg | read name | write name | write range (host-checked) |
|---|---|---|---|
| 00 | RSecond | WSecond | 0–59 |
| 01 | RMinute | WMinute | 0–59 |
| 02 | RHour | WHour | 0–23 |
| 03 | RDayOfWeek | WDayOfWeek | 1–7 |
| 04 | RDayOfMonth | WDayOfMonth | 1–31 |
| 05 | RMonth | WMonth | 1–12 |
| 06 | RYear | WYear | 0–99 |
| 07–0A | RAiSecond/Minute/Hour/DateDay | WAi… | 0–256 (alarm 1) |
| 0B–0D | RAi2Minute/Hour/DateDay | WAi2… | (alarm 2) |
| 0E | RRtcControl | WRtcControl | — |
| 0F | RRtcStatus | WRtcStatus | — |

- Single read: `FA 01 rr 01 01 00 FB` → reply `FA 01 rr 01 01 VV FB`, where VV is BCD.
- Single write: `FA 01 rr 02 01 VV FB`, where VV is BCD of the argument.
- `WAlarmEnable` = `FA 01 0E 02 01 06 FB` and `WAlarmDisable` = `FA 01 0E 02 01 04 FB`. These write RTC control register 0x0E with 0x06 or 0x04.
- Block read `RDateAndTime` = `FA 01 00 03 01 00 FB`. Block read `RMDateAndTime` = `FA 01 00 05 01 00 FB` (alarm block).
  The reply is 13 bytes: `FA 01 00 03 07 SS MM HH DW DD MO YY FB`. All fields are BCD except DW, which is 1..7 with 1 = Sunday.
- Block write `WDateAndTime year mon dom dow hour min sec`:
  `FA 01 00 04 07 SS MM HH DW DD MO YY FB` (13 bytes, BCD, same field order as the read).
- Alarm write `WMAlarmDateAndTime mon dom hour min sec`:
  `FA 01 00 06 05 SS MM HH DD MO FB` (11 bytes, BCD).
- The vendor `rtc` tool sends `WDateAndTime` with year as two digits (0–99) and dow as 1–7 with Sunday = 1. It reads the result back from xmldb `/runtime/rtc` as `"Y/M/D Dow h:m:s"` [verified]. Default time on corruption is 2014-01-01 [from strings].
- When an RTC alarm fires, the MCU sends `FA 20 00 00 xx xx FB` (§4). The vendor then runs `schedule_poweron -s`.

### 3.4 OLED / front-panel display (class 0x04 and 0x55)

**String frames** [verified]: `FA 04 cc ss LEN <LEN ASCII bytes, no NUL> FB`, with a total length of LEN + 6.

| Name | cc ss | Content |
|---|---|---|
| `SysName "str"` | 01 00 | Host name (vendor: Samba NetBIOS name) |
| `FirmwareVer "str"` | 08 00 | Firmware version string |
| `FanSpeed "str"` | 83 00 | **Display only**: `fan_control` sends `"<rpm>rpm"`, for example `FanSpeed 1830rpm`. This does **not** set the fan. |
| `HDHealthy "str"` | 84 00 | Display string |
| `CapacityFree "str"` | 85 00 | Display string |

**Other display commands** [verified]:

| Name | Frame | Notes |
|---|---|---|
| `SysIP1 a.b.c.d` / `SysIP2 …` | `FA 04 02 01/02 08 n0 n1 n2 n3 n4 n5 n6 n7 FB` (14 bytes) | 8 nibbles of the IPv4 address as the 32-bit value `a.b.c.d`, **least-significant nibble first**: `n0 = d&15, n1 = d>>4, n2 = c&15, n3 = c>>4, n4 = b&15, n5 = b>>4, n6 = a&15, n7 = a>>4`. For example 192.168.1.10 → `0A 00 01 00 08 0A 00 0C`. |
| `SysIP1Dis` / `SysIP2Dis` | `FA 04 02 04 00 00 FB` / `FA 04 02 08 00 00 FB` | Clears the IP line |
| `SysIPBondSet` / `SysIPBondClean` | `FA 04 02 10 08 00 FB` / `FA 04 02 20 00 00 FB` | |
| `HDDExist m` | `FA 04 03 b0 b1 b2 b3 FB` (8 bytes) | `bi = (m >> i) & 1` (bay present). Also writes the mask m to `/sys/module/gpio/parameters/hdd_status`. |
| `HDDExist m r` | same 8-byte layout | `bi = ((m>>i)&1) \| (((r>>i)&1) << 4)`, where r marks bays that are red/failed. `hdd_status` = `m & ~r`. |
| `VolRaidState v x` / `VolUsedPct v x` | `FA 04 04 02/03 00 vv xx FB` (8 bytes) | Volume number 0–9 and value (percent 0–100 for UsedPct) [verified layout; argument semantics from strings] |
| `VolStatus …` | `FA 04 04 01 …` variable | Usage: `num id status size valid`, with status codes 0 Normal, 1 Rebuilding, 2 Degraded, 3 Crashed, 4 Lock, 5 Resyncing, 6 Formatting, 7 Resizing, 8 Expand, 9 Migrate. **Byte layout not fully decoded.** |
| `VolClean` | `FA 04 04 04 00 00 FB` | |
| `Standby m` | `FA 04 06 xx 00 00 FB` | m = 0..5 → xx = `1<<m` (0 = 10 s, 1 = 30 s, 2 = 1 min, 3 = 5 min, 4 = 10 min, 5 = 15 min); m = 6 → xx = 0 (disable) [values from strings]. Most likely the OLED/standby timeout. |
| `OledTest t` | `FA 04 07 tt 00 00 FB` | 0 clean, 1 all pixels on, 2 all pixels off |
| `temperature c f` | `FA 04 09 00 00 c_lo c_hi f_lo f_hi FB` (10 bytes) | Little-endian 16-bit °C and °F for the display. `fan_control` sends `(T, T*9/5+32)` whenever T changes. |
| `usbExist v` / `backupStatus v` / `backupDirection v` | `FA 04 0A/0B/0C vv 00 00 FB` | Direction: 0 = USB→NAS, 1 = NAS→USB [from strings] |
| `MFGLCD 0` | `FA 04 72 00 00 00 FB` | Factory LCD test: clear |
| `MFGLCD 1\|2 "str"` | `FA 04 72 01/02 LEN str FB` | Test string on line 1 or 2. The host enforces a **maximum of 16 characters**, which suggests a 16-character line [inferred]. |
| `LCDSleepMode v` | `FA 04 82 vv 00 00 FB` | 0/1 = off/on [from strings]. `up_send_init_info` sends 0 or 1 based on xmldb `/system_mgr/lcd`. |

**Alert line (`ErrStatus code en [string…]`), class 0x55** [verified]:

- Without a string, or when clearing: `FA 55 <idx> <en> FB` (5 bytes).
- With a string: `FA 55 <idx> 01 LEN <ascii> FB`. Codes marked "S" in the vendor usage text require a string. For 2209, 2210 and 2223 the string is the concatenation of argument 3 (a 1-character volume number) and argument 4.
- The host translates the WD alert code into an MCU index:

| WD code → idx | | | |
|---|---|---|---|
| 1 → 25 (sys over temp) | 2 → 26 (sys under temp) | 3 → 30 | 4 → 27 |
| 5 → 31 (pending thermal shutdown) | 6 → 52 | 29 → 9 (fan not working) | 31 → 21 (on UPS power) |
| 36 → 51 | 42 → 48 | 43 → 49 | 44 → 50 |
| 146 → 46 | 201 → 0 | 207 → 1 | 208 → 4 |
| 212 → 17 | 213 → 18 | 224 → 47 | 1001 → 59 |
| 1002 → 23 | 1003 → 14 | 1004 → 32 | 1011 → 11 |
| 1022 → 40 | 1028 → 33 | 1037 → 34 | 1120 → 57 |
| 1121 → 56 | 1124 → 60 | 1127 → 12 | 1130 → 15 |
| 1200 → 44 | 1400 → 55 | 2020 → 52 | 2026 → 13 |
| 2032 → 10 | 2033 → 54 | 2038 → 42 | 2039 → 43 |
| 2040 → 53 | 2126 → 24 | 2128 → 35 | 2129 → 36 |
| 2131 → 37 | 2209 → 5 | 2210 → 6 | 2211 → 7 |
| 2214 → 19 | 2215 → 45 | 2216 → 8 | 2217 → 3 |
| 2218 → 28 | 2222 → 38 | 2223 → 39 | 3000 → 29 |

Code 0, or any code not in this table, is rejected by the host. The meaning of each code is in the vendor usage text. For example 0001 is "System over temperature", 0029 is "Fan Not Working" and 2020 is "System is shutting down".

### 3.5 Commands referenced by vendor code but absent from the table

`FanStatusGet` (used by `libtemperature.so`'s unused UART fan-state path) and `PowerUSB` (`usb_power.sh`) are **not** in `up_send_ctl`'s table, so on this model they only print "Can not find specified command" [verified]. Do not implement them.

### 3.6 Side effect of DeviceShutdown/MCUReset: arming PHY Wake-on-LAN [verified, register meaning inferred]

Before sending the frame, the host programs both Gigabit PHYs (ports 0 and 1) through the Marvell PHY page register 22:

1. Page 17: reg 16 |= 0x5000.
2. Page 3: reg 18 |= 0x0880.
3. Page 0: reg 18 |= 0x0080.

This matches the 88E1318/88E1518 family's WOL control (magic packet enable), LED/INT pin function and WOL interrupt enable [inferred]. The MCU presumably watches the PHY's INT/WOL pin while the SoC is off. In MoodNAS, use `ethtool -s ethX wol g` (if the kernel PHY driver supports it), or replicate these writes through MDIO before `DeviceShutdown`.

---

## 4. MCU → host frames (responses and unsolicited events)

These come from `up_read_daemon`'s 38-entry table [verified]. The "Match" column shows which bytes are compared. "→ key" is where the vendor stores the result in xmldb (shown only as a reference for behaviour).

| Frame (7 bytes unless noted) | Match | Meaning | Vendor action |
|---|---|---|---|
| `FA 01 rr 01 01 VV FB` | b1..b3 | RTC register rr value (BCD) | `"<Name>:<dec>"` → `/runtime/rtc…` |
| 13 bytes `FA … FB` | length only | Date/time block: `SS MM HH DW DD MO YY` (BCD; DW 1 = Sun) | `"YY/MO/DD Dow HH:MM:SS"` → `/runtime/rtc` |
| 15 bytes `FA 03 05 00 xx d0 … d8 FB` | b1..b3 | MCU firmware version: nine data bytes. The vendor formats them as `"%d.%02d.%02d%02d.20%02d"` from `d0+1`, `d1*10+d2`, `d3*10+d4`, `d5*10+d6` and `d7*10+d8` (each byte is a decimal digit) [verified format; semantics inferred] | → `/tmp/up_version` |
| `FA 20 00 00 xx xx FB` | b1..b3 | **RTC alarm fired** | runs `schedule_poweron -s` |
| `FA 30 00 00 xx xx FB` | b1..b3 | **ACK** | `"ack"` → `/runtime/mcu/ack` |
| `FA 21 00 00 …` | b1..b3 | Power button (short press, normal mode) | `"Button:PWR"` → `/runtime/button` |
| `FA 22 00 00 …` | b1..b3 | Reset button | `"Button:RST"` → `/runtime/button` |
| `FA 52 00 00 …` | b1..b3 | Reset button, second variant (long press?) [inferred] | `"Button:RST2"` → `/runtime/button` |
| `FA 57 00 00 …` | b1..b3 | Power button, second variant (long press?) [inferred] | `"Button:PWR2"` → `/runtime/button` |
| `FA 23/24/25/27 00 00 …` | b1..b3 | PWR/RST/FUNC/FUNC2 **pushed** (raw edge events, used in MFG mode) | `"Button:PWRPush"` etc. → `/runtime/MFG` |
| `FA 2A/2B/2C/2D 00 00 …` | b1..b3 | PWR/RST/FUNC/FUNC2 **released** | `"Button:PWRRelease"` etc. → `/runtime/MFG` |
| `FA 26 00 00 …` | b1..b3 | USB backup (copy) button pushed | `"Button:UsbBackupPush"` → `/runtime/usbBackup` |
| `FA 03 08 ss xx RR FB` | b1..b2 | Thermistor reading. ss = 1 → SATA board, 2 → main board. RR = raw ADC byte → °C through the table in §6.3 | → `/runtime/themister/sata` or `/main` |
| `FA 40 VV …` | b1 | "PWR_LOST": probably the power-on cause or a core-power-lost flag (decimal VV) [inferred] | → `/runtime/pwrcorelost` |
| `FA 41 VV …` | b1 | APR (auto power recovery) state (hex VV) | → `/runtime/aprstat` |
| `FA 42 VV …` | b1 | WOL state (hex VV) | → `/runtime/wolstat` |

Unmatched 7-byte frames are dropped silently.

---

## 5. IPC architecture (who actually touches the tty)

```
up_send_ctl (CLI) ──msgsnd──► SysV msg queue ──msgrcv──► up_send_daemon ──write──► /dev/ttyS1 ──► MCU
                                                                               (waits for "ack" in xmldb)
MCU ──► /dev/ttyS1 ──read──► up_read_daemon ──xmldbc_set──► WD xmldb (/runtime/...)  ◄── consumers poll
```

- **Only `up_send_daemon` writes and only `up_read_daemon` reads** the tty during normal operation [verified]. `mcu_upgrade` kills both daemons before it takes over the port.
- **Queue setup** [verified]: `up_send_daemon` calls `ftok(".", 'M')` (relative to its working directory). It then calls `msgget(key, 0)`, or `msgget(key, IPC_CREAT|IPC_EXCL|0660)` if that fails. It writes `"<qid> <pid>\n"` to `/tmp/msg_info.txt`.
- **Message format** [verified]: `long mtype` = the daemon PID read from `/tmp/msg_info.txt`. `mtext[0]` = frame length N and `mtext[1..N]` = the raw frame starting at `FA`. `msgsnd(qid, msg, N+1, 0)`. `up_send_ctl` retries `msgsnd` up to 11 times, re-reading the info file each time.
- **Replies are asynchronous**: `up_send_ctl` never reads the tty. For queries, callers send the command and then poll xmldb. For example `libtemperature`'s main-board read runs `up_send_ctl GetThemister 2`, waits 300 ms, then reads `/runtime/themister/main`, with up to 3 tries.
- **Boot order** (`system_init`) [from strings]: `up_read_daemon &`, `up_send_daemon &`, … `rtc -b`, … `fan_control b c` then `fan_control 0 c &`, … `up_send_ctl MCUVer`. `up_send_init_info` sends `SysName`, `APREnable`/`APRDisable`, `SysIP1`/`SysIP2` or the bond variants, `FirmwareVer`, `backupDirection`, `DeviceReady`, `usbExist 1`, `MCUVer`, and `LCDSleepMode 0|1`.

**MoodNAS recommendation:** run a single Go daemon that owns `/dev/ttyS1`, with one reader goroutine (frame resync on `FA`) and one serialized writer (ACK wait plus retry). Expose a local API (unix socket or D-Bus) instead of SysV queues and xmldb.

---

## 6. Fan control policy (`fan_control`, `libtemperature.so`)

### 6.1 Actuation [verified]

- `LIB_Set_Fan_Speed(n)` runs `up_send_ctl FanControl n` and then sleeps 3 s. Levels 0..4 map 1:1 to the MCU `FanControl` byte. Labels: 0 STOP, 1 LOW, 2 MEDIUM, 3 HIGH, 4 MAX [from strings].
- At start-up the daemon sets **level 1 (LOW)**. The automatic policy never selects level 0; only manual `fan_control -f 0` does.
- Fan-alive detection does not use the UART. `LIB_Get_Fan_State` clears and then polls bit 7 of SoC register offset `0x10154`, for up to 5 × 10 ms. On Kirkwood that offset is the GPIO-high interrupt-cause register, so bit 7 = **GPIO 39 = fan tach edge** [verified offset/bit, GPIO naming inferred]. If the fan does not spin, the daemon raises alert 29 ("Fan Not Working"), runs `led fan_not_working 1`, and touches `/tmp/fan_no_exist`.
- Fan RPM comes from `getFanrpm()` in a library that is not in this set. It is only shown on the OLED (`FanSpeed "<n>rpm"`).

### 6.2 Control temperature [verified]

- **Board temperature** = main-board thermistor (`GetThemister 2`, °C after the table lookup).
- **HDD temperature** = highest SMART attribute 194 across the drives (`smartctl -A -d ata /dev/sdX`).
- **Control temperature T**:
  - If no HDD reading is available (HDD temperature = 0): T = board − 10 if board > 20, otherwise T = board.
  - Otherwise T = max(HDD temperature, board − 10). Precisely: if hdd < board − 9 then T = board − 10, else T = hdd.
- Debug override: if `/tmp/test_temperature` exists, its value is used.
- The main loop period is about **60 s**.

### 6.3 Fan level selection (with hysteresis) [verified]

The policy keeps two candidate levels, r6 and r7, and uses **level = max(r6, r7)** (only when T > 0). The thresholds depend on the current level L:

| Current L | Go up when | Go down when |
|---|---|---|
| 1 | T ≥ 62 → 2 | — |
| 2 | T ≥ 64 → 3 | T ≤ 57 (r6) and T ≤ 58 (r7) → 1 |
| 3 | T ≥ 66 → 4 | T ≤ 60 → 2 |
| 4 | — | T ≤ 61 (r7) and T ≤ 62 (r6) → 3 |

(These thresholds are "HDD-limit offset 10 + constants 47…55" and "65 − T headroom" checks. The HDD-limit helper always returns 10.)

### 6.4 Thermal protection state machine [verified]

States: under-temperature, normal (initial), over-temperature, pending shutdown.

- normal → **over** when T > 64: alert 1 ("System over temperature") and `led temperature_over 1`.
- normal → **under** when T ≤ 0: alert 2. under → normal when T > 1.
- over → normal when T ≤ 64: `led temperature_over 0`, alert cleared (`ErrStatus 1 0`).
- over → **pending shutdown** when T > 69:
  - sets `/runtime/pending_thermal_shutdown`
  - raises alert 5 and creates `/tmp/temperature_over`
  - kills user processes (`kill_running_process`) and enters "safing mode"
- In pending shutdown, every loop with T > 69 increments a counter. At **60** the daemon requests system shutdown through WD's IPC layer, which is about an hour at the 60 s loop period [period inferred]. If T drops to ≤ 69 the state returns to over and the counter resets.
- `up_send_ctl temperature C F` updates the display whenever T changes.

Safe MoodNAS defaults: replicate the 62/64/66 °C up-thresholds with about 4 °C hysteresis. Keep a minimum level of 1. Force level 4 at T ≥ 66. Shut down after sustained T > 69. Treat a lost UART temperature reading as "unknown → level 3 or 4".

### 6.5 Thermistor ADC → °C table [verified]

The table comes from `up_read_daemon`: 192 entries, indexed by the raw byte RR. Values are °C. 127 means "≥127 / sensor fault?". Indices ≥ 0xC0 fall outside the table in the vendor code; treat them as invalid.

```
0x00: 127 127 127 127 127 127 127 127 127 127 127 127 127 125 122 119
0x10: 116 113 110 108 106 103 102 100  98  96  94  93  91  90  88  87
0x20:  85  84  83  82  81  80  79  77  76  75  74  73  72  71  70  70
0x30:  69  68  67  66  65  65  64  63  62  62  61  60  59  59  58  57
0x40:  57  56  55  55  54  53  53  52  52  51  51  50  49  49  48  48
0x50:  47  47  46  46  45  44  44  43  43  42  42  41  41  41  40  40
0x60:  39  39  38  38  37  37  36  36  35  35  34  34  33  33  32  32
0x70:  32  31  31  30  30  30  29  29  28  28  27  27  27  26  26  26
0x80:  25  25  24  24  23  23  23  22  22  21  21  20  20  20  19  19
0x90:  19  18  18  17  17  16  16  16  15  15  14  14  13  13  13  13
0xA0:  12  12  11  11  10  10  10   9   9   8   8   7   7   7   6   6
0xB0:   5   5   4   4   3   3   3   2   2   1   1   0   0   0   0   0
```

This is NTC-style behaviour: a higher raw value means a cooler reading.

---

## 7. I2C: bay power and presence (`libhwshare.so`, `sata_pwr_ctl`)

All access goes through `i2cget`/`i2cset` on **bus 0, address 0x72** [verified]. The register map looks like a PCA9554-style 8-bit I/O expander: reg 0 = input, reg 1 = output [inferred].

| Reg | Bits | Meaning | Evidence |
|---|---|---|---|
| 0x01 (output) | 7 = bay 1, 6 = bay 2, 5 = bay 3, 4 = bay 4 | **SATA power enable, 1 = on**. `LIB_Control_SATA_Power(n, on)` reads reg 1 and then sets or clears bit `0x80 >> (n-1)`. | [verified] |
| 0x01 | 3..0 | Written as 1. A restart writes `((~mask<<4)&0xF0)\|0x0F`, waits 3 s, then writes 0xFF ("all on") and waits 5 s. | [verified] |
| 0x00 (input) | 3 = bay 1, 2 = bay 2, 1 = bay 3, 0 = bay 4 | **Drive present, active-low** (the value is inverted before testing). It is combined with the `/runtime/sata_power` mask unless mode = 2. | [verified] |

- `sata_pwr_ctl -n <1..4> -p <0|1>` switches one bay. Running it without arguments powers off bays whose drive was hot-unplugged [from strings].
- `HWLIB_Set_Auto_Power_Recovery("1")` runs `up_send_ctl APREnable`, otherwise `APRDisable` [verified].

**Temperature sensor at 0x4c:** no binary in this set accesses I2C address 0x4c. `libtemperature.so` contains generic SMBus helpers (`i2cget/i2cset -y <bus> 0x<addr> <reg>`), and `fan_control` has LM75-style `-L/-H` (THYST/TOS) options. However, `LIB_Write_Temperature` is a stub ("not support"), and `LIB_Read_Temperature` uses the **UART thermistor**. On this firmware the board temperature comes from the MCU, not from an I2C sensor [verified]. Whether a 0x4c sensor physically exists must be checked with `i2cdetect -y 0` on hardware.

---

## 8. LEDs driven directly by the SoC (`led`)

`led` uses `LIB_Read_CPU_Register`/`LIB_Write_CPU_Register` with offsets that look like Marvell internal-register offsets. On Kirkwood the internal registers are normally at 0xF1000000 [inferred]. All updates are read-modify-write [verified].

Kirkwood GPIO register offsets [standard SoC map]: low bank `0x10100` data-out, `0x10108` blink-enable. High bank `0x10140` data-out, `0x10148` blink-enable, `0x10154` interrupt cause.

| LED | Register / bit | on | off | blink | Evidence |
|---|---|---|---|---|---|
| Power LED colour A (likely **blue**) | out `0x10100` / blink `0x10108`, bit 28 (GPIO 28) | out\|=bit, blink&=~bit | out&=~bit, blink&=~bit | out\|=bit, blink\|=bit | [verified regs/bits; colour inferred: steady-on is selected on the "system ready" path] |
| Power LED colour B (likely **red**) | same registers, bit 29 (GPIO 29) | same | same | same | [verified regs/bits; colour inferred] |
| HDD 1..4 status LED (likely **red**) | out `0x10140` / blink `0x10148`, bits 13/14/15/16 (GPIO 45/46/47/48) for hd0..hd3 | out\|=bit, blink&=~bit | out&=~bit, blink&=~bit | blink\|=bit | [verified regs/bits; colour inferred] |

The vendor helpers always switch the two power-LED bits as a pair. For example "A blinking" also forces B off. Both off = LED dark.

Other LED paths:

- **Blue activity LEDs and bay "enable" LEDs** appear to be handled by the MCU (`HddAccessLed`, `HDDExist`) and by a kernel module parameter `/sys/module/gpio/parameters/hdd_status` (bay mask) [verified writes; behaviour inferred].
- **`LIB_Write_Mem(0xE00104F0, …)`**: the `led` per-disk error path writes **0x0F44** (off) or **0x007C** (on) to physical address **0xE00104F0**. The value is the same for disks 0–3. The address is outside the SoC register window. It may belong to a PCIe device BAR or be a leftover from another model. **Unknown; do not replicate blindly.**
- **SoC-family caveat:** some vendor scripts (`shutdown.sh`, `usb_power.sh`) poke offsets `0x18004`, `0x18100`, `0x18104`, `0x18140` and `0x18318`. That is the Armada-370/XP MPP/GPIO layout, not Kirkwood. These scripts are shared across WD models. The `led` and `libtemperature` binaries use the Kirkwood `0x101xx` layout. Before porting any register poke, confirm the SoC with `/proc/cpuinfo` and the device tree [inferred].

---

## 9. Open questions and how to confirm on hardware

Run these on the **factory firmware** over SSH. Do not send `DeviceShutdown`, `MCUReset`, `FanControl 0` or anything from `mcu_upgrade` while testing.

1. **Wire capture**:
   - Run `strace -f -tt -x -s 64 -e trace=read,write -p $(pidof up_send_daemon) -p $(pidof up_read_daemon)`.
   - In another shell run `up_send_ctl GetThemister 2`, `up_send_ctl MCUVer`, `up_send_ctl RDateAndTime`, `up_send_ctl WOLStat`, `up_send_ctl APRStat` and `up_send_ctl GetOnCause`.
   - This confirms the byte order in §3 and the ACK frame. It also shows whether data replies and ACKs are separate frames, and gives the exact reply layouts and the meaning of `buf[4]` for 0x40/0x41/0x42 and GetThemister.
2. **Button events**: keep the strace running, press power, reset and USB-copy briefly and long. Record the `0x21/0x22/0x52/0x57/0x26/0x23…` frames and their trailing bytes.
3. **Termios**: run `stty -F /dev/ttyS1 -a` while the daemons run. Expect 19200, cs8, -parenb, -cstopb, -crtscts, raw. Note the actual `min`/`time` values.
4. **Message queue**: run `cat /tmp/msg_info.txt; ipcs -q` to confirm the qid/pid pairing.
5. **Standalone test** (stop both daemons first; the vendor firmware will then miss events):
   ```sh
   stty -F /dev/ttyS1 19200 cs8 -parenb -cstopb -crtscts raw -echo min 1 time 0
   (xxd -c 7 /dev/ttyS1 &) ; printf '\xfa\x03\x08\x02\x00\x00\xfb' > /dev/ttyS1
   ```
   This should produce an ACK `fa 30 …` plus a `fa 03 08 02 …` reply.
6. **Fan levels**: try `FanControl 1..4` and measure RPM. Check whether level 0 really stops the fan and whether the MCU has its own fail-safe if the host goes silent. Watchdog behaviour is unknown.
7. **Shutdown semantics**: find out what the argument `t` of `DeviceShutdown` does (delay in seconds, or a flag?). `shutdown.sh` uses 1 on "MCU shutdown/reboot" models and 0 on OLED models. Also find out what the MCU does after `MCUReset`.
8. **I2C**: run `i2cdetect -y 0` and `i2cdump -y 0 0x72` with bays empty and populated to validate the bit maps in §7, and check whether 0x4c exists.
9. **LED colours**: toggle GPIO 28/29 and GPIO 45–48 (for example with `devmem` at `0xF1010100` and `0xF1010140`) to confirm which colour each bit drives. Investigate `0xE00104F0`.
10. **Not decoded**: the `VolStatus` byte layout, `HDHealthy`/`CapacityFree` display semantics, the exact meaning of the RTC control and status bits (0x06 = alarm enable, 0x04 = alarm disable), and the `mcu_upgrade` ISP protocol (strings: "WT61P8", "Enable ISP", flash 0x0000–0xBFFF, 9600/57600/115200 baud). The ISP protocol is out of scope and risky.
