# WD My Cloud EX4 hardware

What we know, where it comes from, and what still needs a real unit.
Sources: WD's GPL release `WDMyCloud_EX4_GPL_v2.12.127_20181219` (kernel
tree and vendor build scripts) and reverse engineering of the final firmware
image (see [firmware-format.md](firmware-format.md) and
[mcu-protocol.md](mcu-protocol.md)).

| Part | Details | Confidence |
|---|---|---|
| SoC | Marvell 88F6282 (Kirkwood), Feroceon ARMv5TE core. Hardware crypto (CESA) and XOR DMA | confirmed (kernel config, userspace) |
| RAM | 512 MiB | **confirmed** (Stage 0) |
| Flash | Samsung NAND 256 MiB (ID 0xEC/0xDA), 128 KiB erase blocks, see layout below | **confirmed** (Stage 0) |
| Boot loader | Marvell U-Boot 1.1.4, `3.5.9.LIGHTNING-4A.01` (Aug 2013). Stored environment has a bad CRC, so it always boots its built-in default: `nand read.e 0xa00000 0x100000 0x500000; nand read.e 0xf00000 0x600000 0x500000; bootm 0xa00000 0xf00000`. Supports `usb`, `ide`, `ext2load`, `fatload`, `tftpboot`, `usbboot`, `diskboot` | **confirmed** (mtd0 dump) |
| SATA | **All four bays** on a PCIe Marvell **88SX7042** (`11ab:7042`, `sata_mv`, Gen-IIE, 4 ports). ata1-4 = sda-sdd = SCSI host0-3 = bays 1-4. The SoC's own SATA ports are unused | **confirmed** (Stage 0) |
| Bay power / presence | I²C bus 0, address 0x72: register 1 bits 7–4 power bays 1–4; register 0 bits 3–0 = drive present (active low) | from disassembly |
| USB | SoC EHCI + Etron EJ168 USB 3.0 (`1b6f:7023`, xHCI) on PCIe port 1 | **confirmed** (Stage 0) |
| Ethernet | 2× GbE, PHY addresses 0 and 1 (`egiga0/1`), bonded as `bond0` by WD's firmware | **confirmed** (Stage 0) |
| Console | UART0, 115200 8N1 (needs opening the case) | confirmed |
| Front panel | OLED, fan, buttons, RTC, power control: all behind an 8051-class MCU on UART1 at 19200 8N1 | from disassembly, see [mcu-protocol.md](mcu-protocol.md) |
| Temperatures | SoC die sensor; board and SATA-board thermistors read through the MCU | confirmed / from disassembly |
| LEDs | power LED on GPIO 28/29, drive LEDs on GPIO 45–48 | from disassembly, colours to confirm |
| GPIOs | 34 fan power, 35 HDD power, 37 USB VBUS (vendor board file) | from vendor source |

## NAND layout (confirmed on a real unit)

| mtd | Offset | Size | Name | Content |
|---|---|---|---|---|
| mtd0 | 0x0000000 | 1 MiB | u-boot | U-Boot 1.1.4 + (invalid) environment |
| mtd1 | 0x0100000 | 5 MiB | uImage | Linux 3.2.40 |
| mtd2 | 0x0600000 | 5 MiB | ramdisk | gzip'd ext2 root |
| mtd3 | 0x0B00000 | 205 MiB | image | `image.cfs` squashfs |
| mtd6 | 0xD800000 | 5 MiB | reserve | UBI |
| mtd4 | 0xDD00000 | 15 MiB | rescue firmware | rescue kernel at +0x800 (U-Boot string `nand read.e 0xa00000 0xDD00800 0x600000`) |
| mtd5 | 0xEC00000 | 20 MiB | config | UBI volume `config` (settings) |

The kernel reports the machine as "Marvell xCat98DX Development Board" (a
Marvell LSP quirk) and the SoC as `MV88F6282-Rev-Unsupported`. The sampled
unit has 2 bad NAND blocks (inside mtd3) and read errors in mtd4, which is one
more reason MoodNAS does not write to the flash without a serial console.

## Booting a new kernel without touching the flash

- **kexec is not available**: `sys_kexec_load` in WD's kernel is the weak
  "not implemented" stub (`CONFIG_KEXEC` off), and `/proc/config.gz` is absent.
- Changing U-Boot's `bootcmd` (for example to boot from USB) requires
  `saveenv`, which writes NAND, or the serial console.
- Therefore, until a serial console is available, MoodNAS runs as a
  **sidecar** on top of WD's firmware (see [install](../install/README.md)),
  and the next step is a kexec **loadable module** built from WD's own
  GPL kernel tree.

## What the vendor kernel looks like

Linux 3.2.40 with Marvell's 2013 LSP, no device tree, built for the generic
`RD-88F6282-A` reference board. WD's board changes live in
`arch/arm/mach-kirkwood/kw_family/boardEnv/mvBoardEnvSpec.{c,h}`. The front
panel, fan, buttons and RTC have no kernel drivers: closed userspace tools
(`up_send_ctl`, `up_read_daemon`, `fan_control`…) talk to the MCU. MoodNAS
replaces them with its own clean-room Go implementation in
`internal/platform/mcu`.

## Mainline Linux

Mainline already supports Kirkwood 88F6282 (`kirkwood-6282.dtsi`) and all
the on-chip blocks we need (`mv643xx_eth`, `sata_mv`, `orion_nand`,
`mv64xxx_i2c`, `kirkwood-thermal`, CESA, XOR). The work for the EX4 is a
board device tree, [board/wd-mycloud-ex4/kirkwood-wd-mycloud-ex4.dts](../../board/wd-mycloud-ex4/kirkwood-wd-mycloud-ex4.dts)
(draft), plus confirming the open points above with the
[Stage 0 inventory](../install/README.md#stage-0-hardware-inventory).
