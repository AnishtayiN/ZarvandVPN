# ZarvandVPN

A high-performance **DNS-tunnel VPN** client suite — Windows & Android.

## Downloads

Grab the latest build from [Releases](https://github.com/AnishtayiN/ZarvandVPN/releases):

| File | Platform | Notes |
|---|---|---|
| `ZarvandVPN-<ver>-x64-setup.exe` | Windows 7–11 (64-bit) | Full installer |
| `ZarvandVPN-<ver>-x86-setup.exe` | Windows 7–11 (32-bit) | Full installer |
| `ZarvandVPN-<ver>-x64.exe` / `x86.exe` | Windows 7–11 | Portable single exe |
| `ZarvandVPN-<ver>-arm64-v8a-release.apk` | Android 7+ | Modern devices |
| `ZarvandVPN-<ver>-armeabi-v7a-release.apk` | Android 7+ | Older devices |
| `ZarvandVPN-<ver>-x86_64-release.apk` | Android 7+ | Emulators / x86 tablets |

## How it works

The Go core opens a **SOCKS5 proxy on `127.0.0.1:18000`** and tunnels traffic over DNS
queries to the configured tunnel servers.

- **Windows** — the launcher embeds the core, offers a full Settings UI and can flip the
  Windows system proxy with one click, so browsers/apps are routed automatically.
- **Android** — a `VpnService` TUN device routes all traffic through `tun2socks`
  (hev-socks5-tunnel) into the core. No manual proxy configuration needed.

## Settings (both apps)

Everything the core supports is exposed in the UI:

- **Tunnel & Security** — tunnel domains, encryption key, method (None / XOR / ChaCha20 /
  AES-128/192/256-GCM), protocol (SOCKS5 / TCP)
- **Resolvers** — one or more DNS server IPs with a balancing strategy
- **Local proxy** — listen IP/port, optional SOCKS5 user/pass authentication
- **Local DNS** — enable, listen IP/port, persistent cache, TTL
- **Performance** — packet duplication (regular & setup), compression for upload/download
  (Off / Zstd / LZ4 / Zlib), compression threshold, RX/TX and tunnel workers
- **MTU** — min/max for upload and download
- **Advanced** — base-encode data, log level (DEBUG / INFO / WARN / ERROR)

Windows stores settings in `%APPDATA%\ZarvandVPN\settings.json` and renders
`client_config.toml` + `client_resolvers.txt` automatically. Android stores them locally
and generates the same config format at connect time.

## Building

CI (`.github/workflows/release.yml`) builds everything — run it via
**Actions → Release → Run workflow** and enter a version (e.g. `1.0.2`).

## License

See [LICENSE](LICENSE).
