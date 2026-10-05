# ZarvandVPN

A high-performance **DNS-tunnel VPN** client suite by the ZarvandVPN project.

## Components

| Component | Platform | Tech | Min OS |
|---|---|---|---|
| `ZarvandVPN-VPN-x64.exe` / `x86.exe` | Windows | Go 1.20 GUI launcher + embedded core | Windows 7 |
| `zarvand_vpn` APK | Android | Flutter UI + VpnService + tun2socks (hev-socks5-tunnel) + Go core | Android 7.0 (API 24) |

## How it works

The Go core opens a **SOCKS5 proxy on `127.0.0.1:18000`** and tunnels traffic over DNS queries
to the configured tunnel server. The Windows launcher embeds the core binary and manages
start/stop with a clean GUI. The Android app creates a TUN device via `VpnService` and routes
all traffic into the core through `tun2socks`.

## Configuration

- **Windows:** `%APPDATA%\ZarvandVPN\client_config.toml` (created on first run, edit `DOMAINS`,
  `ENCRYPTION_KEY`, `DATA_ENCRYPTION_METHOD`, ... to match your server).
- **Android:** set domains / encryption key in the app; config is generated automatically.

## Build & Release

Releases are built by GitHub Actions:

1. Go to **Actions → Release → Run workflow**
2. Enter the version (e.g. `1.0.0`)
3. Windows exe (x64 + x86, Win7–11) and Android APKs (arm64, armv7, x86_64) are attached to the release.

## Local build (core only)

```bash
go build -o zarvand-client ./cmd/client
./zarvand-client -config client_config.toml
```

## License

See [LICENSE](LICENSE).
