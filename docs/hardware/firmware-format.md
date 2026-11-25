# WD My Cloud EX4 firmware image format (`My_Cloud_LT4A_*.bin`)

Reconstructed from WD's final public image, `My_Cloud_LT4A_2.13.108.bin`
(114,450,460 bytes, SHA-256 `db1ee3e7…74401aa0`). `tools/wdfw` implements
this format and reproduces that image byte for byte (`unpack` + `pack`).

The image is **neither encrypted nor signed**. Integrity is checked only by
XOR checksums.

All integers are 32-bit little-endian. "XOR32" means the XOR of all 32-bit
little-endian words of a region (zero-padded to a multiple of 4).

## Layout

```
0x0000000  header (128 bytes)
0x0000080  kernel   uImage  "Linux-3.2.40", load/entry 0x8000, uncompressed zImage
0x034C920  ramdisk  uImage  "Ramdisk", load 0xE00000, gzip -> 56 MiB ext2 root
0x0536FA8  image.cfs        0x800 sub-header + squashfs 4.0 (xz, 128 KiB blocks), padded to 4 KiB
0x6D167A8  config           gzip'd tar of default settings (default/…)
0x6D1A734  add-on           descriptor + 128-byte header + payload (uP.bin, front-panel MCU firmware)
```

## Header (0x00–0x7F)

| Offset | Field |
|---|---|
| 0x00 / 0x04 | kernel offset / size |
| 0x08 / 0x0C | ramdisk offset / size |
| 0x10 / 0x14 | image.cfs offset / size |
| 0x18 / 0x1C | config offset / size |
| 0x20 … 0x2C | XOR32 of each of the four sections, same order |
| 0x30 | `55 AA` + model (8 bytes, `"Lightni\0"`) + `55 AA` |
| 0x3C | product ID `00 14 00 01` |
| 0x40 | flag byte (`01`: an add-on follows) |
| 0x41 | version string, NUL-terminated (`2.13.108.0109.2022`) |
| 0x78 | header checksum: chosen so the XOR32 of all 32 header words is 0 |
| 0x7C | end of the main image = offset of the first add-on |

## image.cfs sub-header (0x800 bytes)

| Offset | Field |
|---|---|
| +0x00 | length of the squashfs, padded to 4 KiB |
| +0x04 | XOR32 of the padded squashfs |

Because the sub-header carries the payload's own XOR, the section XOR stored
in the main header always equals the padded length.

## Add-on

A 0x60-byte descriptor followed by the add-on data:

| Offset | Field |
|---|---|
| +0x00 | destination directory, 32 bytes (`/tmp`) |
| +0x20 | file name, 32 bytes (`uP.bin`) |
| +0x40 | mode (0x2F3 = 0755) |
| +0x50 | absolute offset of the add-on data |
| +0x54 | add-on length |
| +0x58 | XOR32 of the add-on data |

The add-on data is itself a 128-byte header in the main-header layout (model
`LiguP`, no header checksum) followed by 8051 code for the front-panel MCU.

## How the vendor firmware boots

U-Boot loads `uImage` and `uRamdisk` from NAND (mtd1, mtd2). The ramdisk's
`rc.sh` attaches the UBI config partition (mtd5) and runs `chk_image`, which
loop-mounts `image.cfs` from mtd3 at `/usr/local/modules` and starts
`system_init`.

## Tools

```sh
python3 -I tools/wdfw/wdfw.py info   My_Cloud_LT4A_2.13.108.bin
python3 -I tools/wdfw/wdfw.py unpack My_Cloud_LT4A_2.13.108.bin work/factory
python3 -I tools/wdfw/wdfw.py pack   work/factory rebuilt.bin --version 2.99.0.MOOD
python3 -I tools/wdfw/wdfw.py mkcfs  rootfs.squashfs image.cfs
```

`pack` re-validates every checksum of the image it writes.
