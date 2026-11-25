# Architecture

```
            browser (Svelte SPA, 8 languages)
                 │  HTTPS, session cookie + CSRF header
┌────────────────▼─────────────────────────────────────────────┐
│ moodd (Go, single static binary, ~13 MB for ARMv5)            │
│                                                               │
│  api ─────────── auth (PBKDF2, TOTP, sessions, throttling)    │
│   │              audit log (append-only JSONL)                │
│   │              state (JSON, atomic writes, 0600)            │
│   │                                                           │
│   ├── ai/assistant ── provider: Claude API │ Ollama (LAN)     │
│   │      tools: read-only run now; changes wait for the user  │
│   │      privacy ledger of everything sent                    │
│   │                                                           │
│  monitor (heartbeat) ──── ai/local: disk risk · sentinel ·    │
│   │                        predictive fan · fill forecast     │
│   │                       mood engine → eyes on UI and OLED   │
│   │                                                           │
│  platform ── sim (scenarios)                                  │
│          └─ linux: /proc, /sys, smartctl, mdstat              │
│                    mcu: front-panel protocol on /dev/ttyS1    │
└───────────────────────────────────────────────────────────────┘
       Samba · NFS · WireGuard · nftables (Buildroot, phase 5)
```

## Packages

| Path | Role |
|---|---|
| `cmd/moodd` | entry point, flags, HTTPS, platform choice |
| `internal/api` | REST API, security headers, static UI, assistant tools |
| `internal/auth` | password hashing, TOTP (RFC 6238), sessions, login throttling |
| `internal/state` | persistent configuration |
| `internal/audit` | security audit log |
| `internal/monitor` | sampling loop: runs local AI, drives fan and front panel |
| `internal/mood` | turns the NAS's condition into an eye expression + reasons |
| `internal/ai/local` | on-device models (no network) |
| `internal/ai/assistant` | conversational assistant, providers, pending actions |
| `internal/platform/*` | hardware abstraction: simulator, Linux, MCU protocol |
| `internal/storage` | mdstat and smartctl parsers |
| `internal/tlsutil` | self-signed certificate |
| `ui/` | web interface (Svelte 5 + Vite), `src/i18n/*.json` |
| `tools/wdfw` | WD firmware image unpack/pack |
| `tools/stage0` | read-only hardware inventory script |
| `tools/ai` | training script for the disk-risk model |
| `board/` | device tree, kernel config, Buildroot external tree |

## Design choices

- **Go, mostly standard library.** Cross-compiles to ARMv5 with no C
  toolchain, and gives one static binary with a small attack surface. The only
  third-party Go dependency is the official Anthropic SDK.
- **UI compiled ahead of time.** The 1.2 GHz ARMv5 CPU only serves static
  files and JSON.
- **The API speaks i18n keys.** Errors and mood reasons are keys plus
  parameters, so the UI translates them and the backend stays
  language-neutral.
- **Every hardware call goes through `platform.Platform`.** The same code runs
  in the simulator, in CI and on the NAS.
- **Safety beats AI.** The fan controller forces 100% at critical
  temperatures whatever the mode. Assistant tools that change anything need an
  explicit click.
