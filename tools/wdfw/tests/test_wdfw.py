"""Tests for wdfw using a synthetic image (no WD firmware needed).

Run: python -I -m unittest discover -s tools/wdfw/tests
Set WDFW_REAL_IMAGE=/path/to/My_Cloud_LT4A_*.bin to also round-trip a real image.
"""
import os
import struct
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import wdfw  # noqa: E402


def build_synthetic(path: Path) -> bytes:
    sections = {
        "kernel": struct.pack(">I", wdfw.UIMAGE_MAGIC) + bytes(60) + b"KERNEL" * 100,
        "ramdisk": b"RAMDISK!" * 50,
        "cfs": None,
        "config": b"CFG" * 4,
    }
    sq = b"hsqs" + bytes(range(256)) * 20
    padded = sq + b"\0" * (-len(sq) % 4096)
    sub = bytearray(wdfw.CFS_SUBHEADER_SIZE)
    struct.pack_into("<II", sub, 0, len(padded), wdfw.xor32(padded))
    sections["cfs"] = bytes(sub) + padded

    raw = bytearray(wdfw.HEADER_SIZE)
    raw[0x30:0x3C] = wdfw.MARK + b"Lightni\0" + wdfw.MARK
    raw[0x3C:0x40] = bytes.fromhex("00140001")
    raw[0x40] = 1
    h = wdfw.Header.parse(bytes(raw))
    h.set_version("1.2.3.TEST")
    body = bytearray()
    pos = wdfw.HEADER_SIZE
    for name in wdfw.SECTIONS:
        data = sections[name]
        h.offsets[name], h.sizes[name], h.sums[name] = pos, len(data), wdfw.xor32(data)
        body += data
        pos += len(data)
    h.finalize(pos)
    blob = bytearray(h.raw) + body
    addon = b"\x02\x00\x10" + b"MCU" * 10 + b"\x01"
    desc = bytearray(wdfw.ADDON_DESC_SIZE)
    desc[0:4] = b"/tmp"
    desc[32:38] = b"uP.bin"
    struct.pack_into("<IIII", desc, 0x40, 0x2F3, 0, 0, 0)
    struct.pack_into("<III", desc, 0x50, len(blob) + wdfw.ADDON_DESC_SIZE, len(addon),
                     wdfw.xor32(addon))
    blob += desc + addon
    path.write_bytes(blob)
    return bytes(blob)


class Xor32(unittest.TestCase):
    def naive(self, data):
        data = data + b"\0" * (-len(data) % 4)
        acc = 0
        for (w,) in struct.iter_unpack("<I", data):
            acc ^= w
        return acc

    def test_matches_naive(self):
        for n in (0, 1, 3, 4, 7, 8, 12, 100, 4095, 70000):
            data = os.urandom(n)
            self.assertEqual(wdfw.xor32(data), self.naive(data), n)

    def test_across_chunks(self):
        data = os.urandom((1 << 20) * 2 + 12)
        self.assertEqual(wdfw.xor32(data), self.naive(data))


class RoundTrip(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def roundtrip(self, image: Path):
        out = self.dir / "unpacked"
        self.assertEqual(wdfw.main(["unpack", str(image), str(out)]), 0)
        rebuilt = self.dir / "rebuilt.bin"
        self.assertEqual(wdfw.main(["pack", str(out), str(rebuilt)]), 0)
        self.assertEqual(rebuilt.read_bytes(), image.read_bytes())
        return out

    def test_synthetic_roundtrip(self):
        img = self.dir / "fw.bin"
        build_synthetic(img)
        out = self.roundtrip(img)
        self.assertTrue((out / "addon0_uP.bin").exists())
        self.assertEqual((out / "rootfs.squashfs").read_bytes()[:4], b"hsqs")

    def test_version_change_keeps_valid(self):
        img = self.dir / "fw.bin"
        build_synthetic(img)
        out = self.dir / "u"
        wdfw.main(["unpack", str(img), str(out)])
        new = self.dir / "new.bin"
        self.assertEqual(wdfw.main(["pack", str(out), str(new), "--version", "9.9.9.MOOD"]), 0)
        h, _, _ = wdfw.load(new)
        self.assertEqual(h.version, "9.9.9.MOOD")

    def test_corruption_detected(self):
        img = self.dir / "fw.bin"
        blob = bytearray(build_synthetic(img))
        blob[0x200] ^= 0xFF  # inside the kernel section
        img.write_bytes(blob)
        with self.assertRaises(wdfw.FormatError):
            wdfw.load(img)

    def test_mkcfs(self):
        sq = self.dir / "r.sq"
        sq.write_bytes(b"hsqs" + b"x" * 5000)
        cfs = self.dir / "image.cfs"
        wdfw.main(["mkcfs", str(sq), str(cfs)])
        data = cfs.read_bytes()
        length, csum = struct.unpack_from("<II", data, 0)
        self.assertEqual(length % 4096, 0)
        self.assertEqual(csum, wdfw.xor32(data[wdfw.CFS_SUBHEADER_SIZE:]))
        # as in the factory image: the sub-header cancels the payload, leaving the length
        self.assertEqual(wdfw.xor32(data), length)

    @unittest.skipUnless(os.environ.get("WDFW_REAL_IMAGE"), "WDFW_REAL_IMAGE not set")
    def test_real_image(self):
        self.roundtrip(Path(os.environ["WDFW_REAL_IMAGE"]))


if __name__ == "__main__":
    unittest.main()
