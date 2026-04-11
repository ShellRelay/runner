# ShellRelay Runner

Lightweight Go binary that connects any machine to [ShellRelay](https://www.shellrelay.com) for browser-based terminal access. No VPN, no SSH keys, no port forwarding.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/ShellRelay/runner/main/install.sh | bash
```

Or download a binary from [Releases](https://github.com/ShellRelay/runner/releases).

Supported platforms: **macOS** (arm64, amd64) · **Linux** (arm64, amd64)

## Quick Start

1. Sign in at [shellrelay.com](https://www.shellrelay.com) and register a server — copy the ID and token.
2. Start the runner:

```bash
shellrelay start <server-id> <token>
```

That's it. Open [shellrelay.com](https://www.shellrelay.com), click **Connect**, and you have a browser terminal.

## Docker

Bake the runner into any Docker image for instant browser terminal access:

```dockerfile
FROM ubuntu:24.04
# ... your app setup ...
COPY --from=ghcr.io/shellrelay/runner /usr/local/bin/shellrelay /usr/local/bin/shellrelay
COPY --from=ghcr.io/shellrelay/runner /usr/local/bin/entrypoint.sh /usr/local/bin/entrypoint.sh
ENTRYPOINT ["entrypoint.sh"]
```

```bash
docker run -e SHELLRELAY_EMAIL=you@example.com \
           -e SHELLRELAY_SERVER_ID=my-container \
           myimage
```

The container self-registers. Check `docker logs` for the claim token, then claim it in the dashboard.

## Commands

```
shellrelay start     Start as a background daemon (recommended)
shellrelay stop      Stop the daemon
shellrelay restart   Restart the daemon
shellrelay status    Check if the daemon is running
shellrelay logs      Tail daemon logs (-f to follow, -n <lines>)
shellrelay run       Run in foreground (no daemon)
shellrelay announce  Self-register with the relay (Docker / headless use)
shellrelay rotate    Rotate the server token and restart
shellrelay sessions  List saved session recordings (asciicast)
shellrelay upgrade   Download and install the latest release
shellrelay daemon    Register/remove login service (launchd/systemd)
shellrelay version   Print version
shellrelay help      Show usage
```

## Build from Source

```bash
git clone https://github.com/ShellRelay/runner.git
cd runner
go build -ldflags "-s -w -X main.Version=$(cat VERSION)" -o shellrelay ./cmd/shellrelay
```

Or use the upgrade script to build and install in one step:

```bash
./upgrade.sh
```

## Configuration

Config is stored at `~/.shellrelay/config`:

```
SHELLRELAY_SERVER_ID=my-macbook
SHELLRELAY_TOKEN=sr_xxxxxxxxxxxxxxxxxxxx
```

The relay URL defaults to `wss://api.shellrelay.com` (compiled into the binary). Override with:
- `--relay wss://your-server.com` flag
- `SHELLRELAY_URL=wss://your-server.com` environment variable

## How It Works

1. The runner opens a WebSocket connection to the ShellRelay relay server.
2. When you click **Connect** in the dashboard, the relay bridges your browser to the runner.
3. The runner spawns a PTY (pseudo-terminal) and streams I/O over the WebSocket.
4. Sessions are recorded locally as [asciicast](https://docs.asciinema.org/manual/asciicast/v2/) files.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [Report a Bug](https://github.com/ShellRelay/runner/issues/new?template=bug_report.md)
- [Request a Feature](https://github.com/ShellRelay/runner/issues/new?template=feature_request.md)

## License

[MIT](LICENSE)

Copyright 2025-2026 ShellRelay
