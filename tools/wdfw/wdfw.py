#!/usr/bin/env python3
"""wdfw - unpack, inspect and repack WD My Cloud EX4 (LT4A / "Lightning") firmware images.

Open replacement for the vendor's closed `merge` and `image_checksum` tools.
Format notes: docs/hardware/firmware-format.md

    wdfw info   FIRMWARE.bin
    wdfw unpack FIRMWARE.bin OUTDIR
    wdfw pack   OUTDIR NEW.bin [--version 2.99.0.MOOD]
    wdfw mkcfs  rootfs.squashfs image.cfs

Only the Python standard library is used.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import struct
import sys
from dataclasses import dataclass, field
from pathlib import Path

HEADER_SIZE = 0x80
CFS_SUBHEADER_SIZE = 0x800
ADDON_DESC_SIZE = 0x60
SECTIONS = ("kernel", "ramdisk", "cfs", "config")
SECTION_FILES = {
    "kernel": "uImage",
    "ramdisk": "uRamdisk",
    "cfs": "image.cfs",
    "config": "default.tar.gz",
}
MARK = b"\x55\xaa"
UIMAGE_MAGIC = 0x27051956


class FormatError(Exception):
    pass


def xor32(data: bytes | memoryview) -> int:
    """XOR of all 32-bit little-endian words (zero-padded to a multiple of 4)."""
    mv = memoryview(data)
    acc = 0
    chunk = 1 << 20
    for start in range(0, len(mv), chunk):
        part = bytes(mv[start:start + chunk])
        if len(part) % 4:
            part += b"\0" * (4 - len(part) % 4)
        x = int.from_bytes(part, "little")
        bits = len(part) * 8
        while bits > 32:
            # fold the integer in half; pad to a whole number of words first
            words = bits // 32
            half = (words + 1) // 2 * 32
            x = (x >> half) ^ (x & ((1 << half) - 1))
            bits = half
        acc ^= x
    return acc & 0xFFFFFFFF


@dataclass
class Header:
    raw: bytearray
    offsets: dict = field(default_factory=dict)
    sizes: dict = field(default_factory=dict)
    sums: dict = field(default_factory=dict)

    @classmethod
    def parse(cls, raw: bytes) -> "Header":
        if len(raw) < HEADER_SIZE:
            raise FormatError("file shorter than the 128-byte header")
        h = cls(bytearray(raw[:HEADER_SIZE]))
        vals = struct.unpack_from("<8I", h.raw, 0)
        for i, name in enumerate(SECTIONS):
            h.offsets[name] = vals[2 * i]
            h.sizes[name] = vals[2 * i + 1]
        for i, name in enumerate(SECTIONS):
            h.sums[name] = struct.unpack_from("<I", h.raw, 0x20 + 4 * i)[0]
        if h.raw[0x30:0x32] != MARK:
            raise FormatError("missing 55 AA model marker at 0x30 - not an LT4A image?")
        return h

    @property
    def model(self) -> str:
        return self.raw[0x32:0x3A].split(b"\0")[0].decode("ascii", "replace")

    @property
    def product_id(self) -> str:
        return self.raw[0x3C:0x40].hex()

    @property
    def version(self) -> str:
        return self.raw[0x41:0x78].split(b"\0")[0].decode("ascii", "replace")

    def set_version(self, v: str) -> None:
        b = v.encode("ascii")
        if len(b) > 0x78 - 0x41 - 1:
            raise FormatError("version string too long")
        self.raw[0x41:0x78] = b + b"\0" * (0x78 - 0x41 - len(b))

    @property
    def header_sum(self) -> int:
        return struct.unpack_from("<I", self.raw, 0x78)[0]

    @property
    def main_end(self) -> int:
        return struct.unpack_from("<I", self.raw, 0x7C)[0]

    def finalize(self, main_end: int) -> None:
        for i, name in enumerate(SECTIONS):
            struct.pack_into("<II", self.raw, 8 * i, self.offsets[name], self.sizes[name])
            struct.pack_into("<I", self.raw, 0x20 + 4 * i, self.sums[name])
        struct.pack_into("<I", self.raw, 0x7C, main_end)
        struct.pack_into("<I", self.raw, 0x78, 0)
        struct.pack_into("<I", self.raw, 0x78, xor32(self.raw))


def parse_uimage(data: bytes) -> dict | None:
    if len(data) < 64 or struct.unpack_from(">I", data, 0)[0] != UIMAGE_MAGIC:
        return None
    _, hcrc, ts, size, load, entry, dcrc, os_, arch, typ, comp = struct.unpack_from(
        ">7I4B", data, 0)
    name = data[32:64].split(b"\0")[0].decode("ascii", "replace")
    return {"name": name, "timestamp": ts, "size": size, "load": hex(load),
            "entry": hex(entry), "type": typ, "compression": comp}


def parse_addons(blob: bytes, main_end: int) -> list[dict]:
    """Trailing add-ons: 0x60 descriptor followed by the add-on payload."""
    addons = []
    pos = main_end
    while pos + ADDON_DESC_SIZE <= len(blob):
        desc = blob[pos:pos + ADDON_DESC_SIZE]
        off, length, csum = struct.unpack_from("<III", desc, 0x50)
        if off != pos + ADDON_DESC_SIZE or off + length > len(blob):
            raise FormatError(f"bad add-on descriptor at {pos:#x}")
        payload = blob[off:off + length]
        if xor32(payload) != csum:
            raise FormatError(f"add-on checksum mismatch at {off:#x}")
        addons.append({
            "dir": desc[0:32].split(b"\0")[0].decode(),
            "name": desc[32:64].split(b"\0")[0].decode(),
            "descriptor": desc.hex(),
            "data": payload,
        })
        pos = off + length
    if pos != len(blob):
        raise FormatError(f"{len(blob) - pos} unexplained trailing bytes at {pos:#x}")
    return addons


def load(path: Path) -> tuple[Header, bytes, list[dict]]:
    blob = path.read_bytes()
    h = Header.parse(blob)
    if xor32(blob[:HEADER_SIZE]) != 0:
        raise FormatError("header checksum invalid")
    for name in SECTIONS:
        o, s = h.offsets[name], h.sizes[name]
        if o + s > len(blob):
            raise FormatError(f"section {name} runs past end of file")
        got = xor32(memoryview(blob)[o:o + s])
        if got != h.sums[name]:
            raise FormatError(f"section {name}: checksum {got:#010x} != {h.sums[name]:#010x}")
    return h, blob, parse_addons(blob, h.main_end)


def cmd_info(args) -> int:
    h, blob, addons = load(Path(args.firmware))
    print(f"file      {args.firmware} ({len(blob):,} bytes)")
    print(f"sha256    {hashlib.sha256(blob).hexdigest()}")
    print(f"model     {h.model}   product id {h.product_id}")
    print(f"version   {h.version}")
    for name in SECTIONS:
        o, s = h.offsets[name], h.sizes[name]
        extra = ""
        ui = parse_uimage(blob[o:o + 64])
        if ui:
            extra = f"  uImage '{ui['name']}' load {ui['load']}"
        print(f"{name:9} @ {o:#010x}  {s:>11,} bytes  xor32 {h.sums[name]:#010x} OK{extra}")
    for a in addons:
        print(f"add-on    {a['dir']}/{a['name']}  {len(a['data']):,} bytes  OK")
    return 0


def cmd_unpack(args) -> int:
    h, blob, addons = load(Path(args.firmware))
    out = Path(args.outdir)
    out.mkdir(parents=True, exist_ok=True)
    for name in SECTIONS:
        o, s = h.offsets[name], h.sizes[name]
        (out / SECTION_FILES[name]).write_bytes(blob[o:o + s])
    cfs = blob[h.offsets["cfs"]:h.offsets["cfs"] + h.sizes["cfs"]]
    sq_len = struct.unpack_from("<I", cfs, 0)[0]
    (out / "rootfs.squashfs").write_bytes(cfs[CFS_SUBHEADER_SIZE:CFS_SUBHEADER_SIZE + sq_len])
    manifest = {
        "format": "wd-lt4a",
        "model": h.model,
        "version": h.version,
        "header": bytes(h.raw).hex(),
        "sections": [SECTION_FILES[n] for n in SECTIONS],
        "gaps": [],
        "addons": [],
        "source_sha256": hashlib.sha256(blob).hexdigest(),
    }
    # bytes between the end of one section and the start of the next (normally none)
    ends = [HEADER_SIZE] + [h.offsets[n] + h.sizes[n] for n in SECTIONS]
    for i, name in enumerate(SECTIONS):
        gap = blob[ends[i]:h.offsets[name]]
        manifest["gaps"].append(gap.hex())
    for i, a in enumerate(addons):
        fname = f"addon{i}_{a['name']}"
        (out / fname).write_bytes(a["data"])
        manifest["addons"].append({"file": fname, "dir": a["dir"], "name": a["name"],
                                   "descriptor": a["descriptor"]})
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n")
    print(f"unpacked {len(SECTIONS)} sections and {len(addons)} add-on(s) to {out}")
    return 0


def cmd_pack(args) -> int:
    src = Path(args.indir)
    m = json.loads((src / "manifest.json").read_text())
    h = Header.parse(bytes.fromhex(m["header"]))
    if args.version:
        h.set_version(args.version)
    body = bytearray()
    pos = HEADER_SIZE
    for i, name in enumerate(SECTIONS):
        gap = bytes.fromhex(m["gaps"][i])
        body += gap
        pos += len(gap)
        data = (src / SECTION_FILES[name]).read_bytes()
        h.offsets[name], h.sizes[name], h.sums[name] = pos, len(data), xor32(data)
        body += data
        pos += len(data)
    h.finalize(pos)
    out = bytearray(h.raw) + body
    for a in m["addons"]:
        data = (src / a["file"]).read_bytes()
        desc = bytearray(bytes.fromhex(a["descriptor"]))
        struct.pack_into("<III", desc, 0x50, len(out) + ADDON_DESC_SIZE, len(data), xor32(data))
        out += desc + data
    Path(args.output).write_bytes(out)
    load(Path(args.output))  # self-check
    print(f"wrote {args.output} ({len(out):,} bytes) sha256 {hashlib.sha256(out).hexdigest()}")
    return 0


def cmd_mkcfs(args) -> int:
    sq = Path(args.squashfs).read_bytes()
    if sq[:4] != b"hsqs":
        raise FormatError("input is not a little-endian squashfs image")
    padded = sq + b"\0" * (-len(sq) % 4096)
    sub = bytearray(CFS_SUBHEADER_SIZE)
    struct.pack_into("<II", sub, 0, len(padded), xor32(padded))
    Path(args.output).write_bytes(bytes(sub) + padded)
    print(f"wrote {args.output}")
    return 0


def main(argv=None) -> int:
    p = argparse.ArgumentParser(prog="wdfw", description=__doc__.splitlines()[0])
    sub = p.add_subparsers(dest="cmd", required=True)
    s = sub.add_parser("info", help="validate and describe an image")
    s.add_argument("firmware")
    s.set_defaults(func=cmd_info)
    s = sub.add_parser("unpack", help="split an image into its sections")
    s.add_argument("firmware")
    s.add_argument("outdir")
    s.set_defaults(func=cmd_unpack)
    s = sub.add_parser("pack", help="rebuild an image from an unpacked directory")
    s.add_argument("indir")
    s.add_argument("output")
    s.add_argument("--version", help="replace the version string")
    s.set_defaults(func=cmd_pack)
    s = sub.add_parser("mkcfs", help="wrap a squashfs into an image.cfs section")
    s.add_argument("squashfs")
    s.add_argument("output")
    s.set_defaults(func=cmd_mkcfs)
    args = p.parse_args(argv)
    try:
        return args.func(args)
    except FormatError as e:
        print(f"wdfw: {e}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())
