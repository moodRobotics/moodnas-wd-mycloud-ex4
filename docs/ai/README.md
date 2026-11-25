# Artificial intelligence in MoodNAS

The EX4 has a 2012-era ARMv5 CPU and 512 MB of RAM. It cannot run a large
language model, and it does not need to. MoodNAS splits the work:

| Feature | Where it runs | Data that leaves the NAS |
|---|---|---|
| Disk failure prediction | on the NAS | none |
| Ransomware sentinel | on the NAS | none |
| Predictive fan control | on the NAS | none |
| Capacity forecast | on the NAS | none |
| Mood (eyes on the UI and OLED) | on the NAS | none |
| Conversational assistant | Claude API, or Ollama on your LAN | your message, plus status data from the tools it calls |
| Explained alerts | provider (planned) | alert text |
| Semantic search, photo tagging | provider (roadmap) | document/photo content, opt-in per share |

On-device features are on by default. Provider features are **off** until an
administrator chooses a provider. Every request sent to a provider is listed
in the **privacy ledger** on the AI page.

## Disk failure prediction

A logistic regression over the SMART attributes that Backblaze and Google
(Pinheiro et al., 2007) found most predictive: reallocated (5), reported
uncorrectable (187), command timeout (188), pending (197), offline
uncorrectable (198), plus age and temperature. Inputs are `log1p`-scaled.

The built-in weights are hand-set priors. To fit real weights, download
Backblaze's public drive-stats data and run:

```sh
python3 -I tools/ai/train_diskrisk.py data_Q1_2025/ data_Q2_2025/ -o diskrisk-model.json
```

Put the file in the data directory (`/var/lib/moodnas`). The UI shows each
disk's probability and the factors behind it.

## Ransomware sentinel

Ransomware rewrites many files with encrypted (high-entropy, near 8
bits/byte) content and often renames them (`photo.jpg.locked`). The sentinel
scores file events per share over a sliding window:

- a high-entropy write to a file whose type is not naturally compressed: 1
  (1.5 with a double extension);
- a rename to an unknown long or double extension: 2;
- a delete: 0.5;
- a high-entropy write to JPEG/MP4/ZIP…: 0.02. Photo imports are not
  attacks.

Above the threshold (30 points per minute) the share is **frozen
read-only**, the incident is audited, the eyes turn red, and the OLED shows
`SHARE FROZEN`. An administrator reviews the incident and unfreezes the share.
Events come from fanotify on the real NAS (phase 5) and from scenarios in the
simulator.

## Predictive fan

The controller fits a linear trend to the last 5 minutes of CPU temperature
and reacts to the temperature expected 3 minutes ahead, so the fan spins up
before the heat arrives. During quiet hours it caps the fan at 45% while
temperatures are safe. At 90 °C CPU or 55 °C disk it goes to 100% whatever
the mode, and hysteresis avoids constant speed changes. On the EX4 the fan
has 5 levels, driven through the front-panel MCU.

## Capacity forecast

Least-squares regression of used bytes over time, giving bytes per day and
days until the volume is full. It warns when the volume will be full in
under 30 days.

## Assistant ("Mood")

- **Providers:** the Claude API, through the official Anthropic Go SDK (model
  configurable, `claude-opus-5-5` by default, with server-side refusal
  fallback), or any Ollama server on your network.
- **Read-only tools** (status, storage, security, shares, users) run
  immediately.
- **Tools that change something** (create a share, fan mode, unfreeze a share)
  create a **pending action**. It runs only when the user clicks *Do it* in
  the UI, and it expires after 15 minutes.
- Regular users get only read-only tools that see their own shares. Admin
  tools are offered only to administrators.
- The system prompt tells the model that file and share names are data, not
  instructions, to resist prompt injection through file names.
- The API key is write-only: it is stored on the NAS (file mode 0600) and the
  API never returns it.
