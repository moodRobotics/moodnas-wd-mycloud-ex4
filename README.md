<p align="center">
  <img src="assets/brand/mood-robotics.png" alt="Mood Robotics" width="360">
</p>

<h1 align="center">MoodNAS</h1>

<p align="center">
  <strong>A new open-source firmware that gives the WD My Cloud EX4 a second life.<br>
  Old hardware, modern security, built-in artificial intelligence.</strong>
</p>

<p align="center">
  <b>English</b> ·
  <a href="README.es.md">Español</a> ·
  <a href="README.pt.md">Português</a> ·
  <a href="README.fr.md">Français</a> ·
  <a href="README.de.md">Deutsch</a> ·
  <a href="README.it.md">Italiano</a> ·
  <a href="README.zh-CN.md">简体中文</a> ·
  <a href="README.ja.md">日本語</a>
</p>

<p align="center">
  <img alt="License: GPL-3.0" src="https://img.shields.io/badge/license-GPL--3.0-1f5fff">
  <img alt="Status: alpha, simulator" src="https://img.shields.io/badge/status-alpha%20·%20simulator-ffb020">
  <img alt="Languages: 8" src="https://img.shields.io/badge/languages-8-2fd27a">
</p>

---

## Why

The **WD My Cloud EX4** is a solid 4-bay NAS that Western Digital no longer
supports. Its last firmware runs Linux 3.2 (2013), OpenSSL 1.0.1 and PHP 5.4,
and ships with a default root password. Thousands of these boxes still hold
family photos and small-business files on networks that are increasingly
hostile.

**MoodNAS** replaces that firmware with a modern, open system. It uses every
part of the hardware and adds something an old ARM box was never expected to
have: **artificial intelligence that protects your data**.

<p align="center">
  <img src="docs/img/dashboard.png" alt="MoodNAS overview" width="900">
</p>

## Highlights

### 👀 A NAS with a mood
The Mood Robotics eyes are MoodNAS's face. They are calm when all is well,
focused when busy, resting at night, worried when something needs attention,
and alarmed (with a red glow) when your data is at risk. The same eyes appear
on the NAS's **front OLED display**, so you can tell how it is doing from
across the room.

### 🧠 AI that runs on the NAS itself
These models need no cloud and send nothing out of your home:

| | |
|---|---|
| **Disk failure prediction** | Reads each disk's SMART data with a model based on Backblaze's public failure statistics. It tells you which disk will probably fail and why, before it happens. |
| **Ransomware sentinel** | Spots mass encryption of files (entropy analysis plus rename patterns) and **freezes the share read-only within seconds**. |
| **Predictive fan** | Anticipates heat from the temperature trend and keeps quiet at night, without ever compromising safety. |
| **Capacity forecast** | Learns how fast you store data and warns weeks before the volume is full. |

### 💬 "Mood", your assistant
Ask in your own words: *"How is my NAS doing?"*, *"Should I worry about any
disk?"*, *"Create a shared folder called Photos for everyone"*. Mood works
with **Claude** (Anthropic API) or with **Ollama on your own network**, so
nothing leaves your home.

- It only looks at what its tools show it. Every change waits for **your
  confirmation**.
- A **privacy ledger** lists every request that left the NAS.

<p align="center">
  <img src="docs/img/ransomware.png" alt="The ransomware sentinel froze a share" width="900">
</p>

### 🔒 Secure by design
- No default passwords: first-time setup requires a one-time code shown on
  the front display.
- HTTPS by default.
- PBKDF2-SHA512 password hashing, two-factor authentication (TOTP), and login
  throttling.
- CSRF protection, a strict Content-Security-Policy, and an audit log.
- One static Go binary with a minimal dependency set.

[More in SECURITY.md](SECURITY.md).

### 🌍 In your language
Interface and documentation in **Spanish, English, Portuguese, French,
German, Italian, Chinese and Japanese**. Each user picks their own language.

### 💾 Your data stays where it is
MoodNAS reads the RAID volumes created by WD's firmware **as they are**. No
formatting, no migration.

## Status

> **MoodNAS is alpha software.** The full interface, the API, the local AI and
> the assistant run today against a **simulated EX4**. Booting on the real
> hardware is the next milestone, and you can help.

| Phase | | State |
|---|---|---|
| 1 | Firmware analysis, open `wdfw` image tool (byte-exact repack) | ✅ done |
| 2 | Hardware inventory script (Stage 0, read-only) | ✅ ready, needs reports from real units |
| 3 | Web UI (8 languages), API, security, simulator | ✅ done |
| 4 | On-device AI + assistant (Claude / Ollama) | ✅ first version |
| 4b | Explained alerts, semantic search, photo tagging | 🔜 planned |
| 5 | Mainline kernel + Buildroot, kexec boot (no flash writes), MCU driver | 🚧 in progress: draft device tree, MCU protocol decoded |
| 6 | Permanent installation | ⏳ needs serial-console recovery path |

## Try it in two minutes

You need Docker:

```sh
git clone https://github.com/moodrobotics/moodnas-wd-mycloud-ex4.git
cd moodnas-wd-mycloud-ex4
docker compose up --build
```

1. Open **http://localhost:8080**.
2. Sign in with the demo account **`demo`** / **`moodnas-demo`**.
3. On the overview page, try the simulator scenarios: heat wave, failing disk,
   degraded RAID, ransomware attack. Watch the eyes.

## First access and passwords

| Where | User | Password |
|---|---|---|
| Demo (`docker compose`, simulator) | `demo` | `moodnas-demo` |
| Your real NAS | **none by default** | **none by default** |

On a real NAS there is **no default user or password**, on purpose: the
original WD firmware shipped with a well-known root password, one of the
weaknesses MoodNAS removes. On first start:

1. The NAS shows an 8-character **setup code** on its front display (and in
   the system log).
2. Open `https://<nas-ip>` and enter the code. This proves you are physically
   next to the NAS, so nobody else on your network can claim it.
3. Create **your own** administrator and password (at least 10 characters).
   You can turn on two-factor authentication afterwards in *My account*.

The demo account exists only with `moodd -simulate -demo`. `moodd` refuses
`-demo` on real hardware, because its password is public.

Forgot the password? Reset procedure: see [docs/install](docs/install/README.md)
(planned with the hardware port).

## How it works

```
Browser (Svelte, 8 languages) ──HTTPS──▶ moodd (Go, one static binary)
                                         ├─ API · auth · audit
                                         ├─ local AI: disk risk · sentinel · fan · forecast
                                         ├─ assistant ─▶ Claude API | Ollama (LAN)
                                         └─ platform ─▶ simulator | EX4 hardware (SATA, SMART, MCU, OLED)
```

| | |
|---|---|
| Architecture | [docs/architecture](docs/architecture/README.md) |
| AI in depth | [docs/ai](docs/ai/README.md) |
| Hardware | [docs/hardware](docs/hardware/README.md): [firmware format](docs/hardware/firmware-format.md) · [front-panel MCU protocol](docs/hardware/mcu-protocol.md) |
| Installation plan and Stage 0 | [docs/install](docs/install/README.md) |

## Hardware

| | WD My Cloud EX4 |
|---|---|
| SoC | Marvell Armada 300 / Kirkwood 88F6282, ARMv5TE, with hardware crypto |
| RAM | 512 MB |
| Bays | 4 × 3.5" SATA |
| Network | 2 × Gigabit Ethernet |
| Front panel | OLED display, buttons, fan via a microcontroller |

Other WD models (EX2, EX4100, PR4100) are not supported yet.
Contributions are welcome.

## Contributing

The most useful help right now is a **[Stage 0 report](docs/install/README.md#stage-0-hardware-inventory)**
from a real EX4. It is read-only and also backs up your flash. See
[CONTRIBUTING.md](CONTRIBUTING.md) for development and translations.

## Legal

MoodNAS is free software under the [GNU GPL v3.0 or later](LICENSE). It is
an independent project by **Mood Robotics** and is not affiliated with,
endorsed or supported by Western Digital. "WD" and "My Cloud" are trademarks
of Western Digital Corporation, used here only to describe hardware
compatibility. This repository contains no Western Digital firmware or
binaries; the tools work on files you download yourself.

Replacing a device's firmware has risks. MoodNAS is provided without
warranty: back up your data first.

<p align="center"><sub>Made with ♥ by Mood Robotics: old hardware deserves new intelligence.</sub></p>
