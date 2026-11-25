# Contributing to MoodNAS

Thank you! The most valuable contributions right now:

1. **Stage 0 reports** from real EX4 units ([how](docs/install/README.md#stage-0-hardware-inventory)).
2. **Hardware bring-up:** device tree, kexec boot, confirming the MCU protocol.
3. **Translations:** `ui/src/i18n/<lang>.json` and `README.<lang>.md`.
4. **AI models:** better disk-risk weights, sentinel tuning with real traces.

## Development

Requirements: Docker, Node.js 20+, Python 3.10+. Go runs in a container via
`build/go.sh`, but a local Go 1.26+ works too.

```sh
make test        # Go vet + tests, Python tests, i18n check
make ui          # build the web UI into ui/dist
make moodd-arm   # static ARMv5 binary for the EX4 in dist/
make demo        # simulator + UI on http://localhost:8080
```

UI live-reload: run `moodd -simulate -insecure-http -listen :8080 -http "" -data work/data -ui ui/dist`,
then `cd ui && npm run dev`.

## Rules

- **Never commit WD firmware, binaries or extracted files.** Tools must work
  on the user's own download. `work/` is git-ignored for this reason.
- **Clean-room reverse engineering.** Document behaviour (formats, constants,
  byte sequences) in `docs/`. Never copy vendor code.
- **Every UI string goes through i18n.** Add the key to all 8 files;
  `npm run check-i18n` enforces it.
- **Safety first.** Anything that writes to flash, the MCU or disk
  partitions needs a review and a recovery story.
- Go code: `gofmt`, standard library first, tests for parsing and security
  logic.

By contributing you agree that your work is licensed under GPL-3.0-or-later.
