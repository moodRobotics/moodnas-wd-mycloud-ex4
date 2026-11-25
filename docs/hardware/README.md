# WD My Cloud EX4 hardware

What we know, where it comes from, and what still needs a real unit.
Sources: WD's GPL release `WDMyCloud_EX4_GPL_v2.12.127_20181219` (kernel
tree and vendor build scripts) and reverse engineering of the final firmware
image (see [firmware-format.md](firmware-format.md) and
[mcu-protocol.md](mcu-protocol.md)).

| Part | Details | Confidence |
|---|---|---|
| SoC | Marvell 88F6282 (Kirkwood), Feroceon ARMv5TE core. Hardware crypto (CESA) and XOR DMA | confirmed (kernel config, userspace) |
| RAM | 512 MiB | not stated in the GPL tree; to confirm with Stage 0 |
| Flash | NAND on the SoC controller. mtd0 U-Boot, mtd1 kernel, mtd2 ramdisk, mtd3 image.cfs, mtd4 rescue firmware + MAC/serial, mtd5 config (UBI) | roles confirmed, sizes unknown |
| Boot loader | Marvell U-Boot 1.1.x; no source in the GPL release | inferred |
| SATA | 2 ports on the SoC (`sata_mv`) + a PCIe Marvell controller (AHCI) | confirmed |
| Bay power / presence | I²C bus 0, address 0x72: register 1 bits 7–4 power bays 1–4; register 0 bits 3–0 = drive present (active low) | from disassembly |
| USB | SoC EHCI + Etron EJ168 USB 3.0 (xHCI) on PCIe | confirmed |
| Ethernet | 2× GbE, RGMII, PHYs at SMI 0 and 1 | confirmed |
| Console | UART0, 115200 8N1 (needs opening the case) | confirmed |
| Front panel | OLED, fan, buttons, RTC, power control: all behind an 8051-class MCU on UART1 at 19200 8N1 | from disassembly, see [mcu-protocol.md](mcu-protocol.md) |
| Temperatures | SoC die sensor; board and SATA-board thermistors read through the MCU | confirmed / from disassembly |
| LEDs | power LED on GPIO 28/29, drive LEDs on GPIO 45–48 | from disassembly, colours to confirm |
| GPIOs | 34 fan power, 35 HDD power, 37 USB VBUS (vendor board file) | from vendor source |

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
