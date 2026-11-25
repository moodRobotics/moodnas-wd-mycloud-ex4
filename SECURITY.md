# Security

MoodNAS exists largely because the original EX4 firmware is no longer
patched. It ran Linux 3.2 (2013), OpenSSL 1.0.1, PHP 5.4, and had a default
root password. Security is a design goal, not a feature.

## Reporting a vulnerability

Please **do not open a public issue**. Use GitHub's *Report a vulnerability*
(private security advisory) on this repository. We aim to answer within 7
days.

## What MoodNAS does today

- **No default credentials.** First-time setup requires a one-time code shown
  on the NAS's front display and in the system log, so nobody else on the
  network can claim a fresh NAS. The public demo account
  (`demo` / `moodnas-demo`) is created only by `moodd -simulate -demo`, and
  `moodd` refuses `-demo` without `-simulate`.
- **HTTPS by default** with a self-signed ECDSA P-256 certificate. Its
  SHA-256 fingerprint is logged so you can check the browser warning. HTTP
  only redirects. HSTS is sent over TLS.
- **Passwords** are hashed with PBKDF2-HMAC-SHA512 (210,000 iterations) and
  must be at least 10 characters. Login takes the same time whether or not the
  user exists.
- **Two-factor authentication** (TOTP, RFC 6238) for every account.
- **Login throttling:** after 5 failed attempts per client address, a lock-out
  that doubles up to 15 minutes.
- **Sessions:** random 256-bit tokens in `HttpOnly`, `Secure`,
  `SameSite=Strict` cookies. 30-minute idle and 12-hour absolute timeouts. All
  sessions end when the password changes.
- **CSRF protection:** a per-session token is required in a header on every
  state-changing request.
- **Strict input handling:** JSON bodies are size-limited and unknown fields
  are rejected. Names are validated against allow-lists.
- **Headers:** a strict Content-Security-Policy (no inline scripts, no third
  parties), `X-Frame-Options: DENY`, `nosniff`, `no-referrer`.
- **Audit log** of logins, setup, user, share and settings changes, and AI
  actions.
- **Least privilege for AI:** assistant actions that change anything need an
  explicit confirmation. The API key is write-only. A privacy ledger records
  every outbound AI request.
- **Ransomware sentinel** freezes a share read-only when it sees mass
  encryption.
- **Supply chain:** Go standard library plus the official Anthropic SDK. The
  UI is built at compile time with no runtime third-party resources.

## Planned (with the hardware port)

Mainline kernel with hardening options, nftables deny-by-default firewall,
WireGuard VPN instead of port forwarding, signed A/B updates (ed25519),
Samba with SMB1 disabled, a sandboxed `moodd` (seccomp/Landlock), and
snapshots for ransomware recovery.
