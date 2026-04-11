# Contributing to ShellRelay Runner

Thank you for your interest in contributing! This guide will help you get started.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- Git
- A Unix-like OS (macOS or Linux)

### Development Setup

```bash
# Clone the repo
git clone https://github.com/ShellRelay/runner.git
cd runner

# Build
go build -o shellrelay ./cmd/shellrelay

# Run tests
go test ./...

# Run vet
go vet ./...
```

### Project Structure

```
cmd/shellrelay/     CLI commands (main, run, start/stop, announce, upgrade, etc.)
internal/relay/      WebSocket connection and PTY management
internal/config/     Config file load/save
internal/session/    Asciicast session recording
install.sh          One-liner installer
Dockerfile          Multi-stage Docker build
entrypoint.sh       Docker entrypoint (manual + announce modes)
```

## How to Contribute

### Reporting Bugs

1. Check [existing issues](https://github.com/ShellRelay/runner/issues) to avoid duplicates.
2. Use the **Bug Report** issue template.
3. Include your OS, architecture, runner version (`shellrelay version`), and steps to reproduce.

### Suggesting Features

1. Open an issue using the **Feature Request** template.
2. Describe the use case and why it would be useful.

### Submitting Pull Requests

1. Fork the repo and create a branch from `main`:
   ```bash
   git checkout -b my-feature
   ```
2. Make your changes.
3. Ensure tests pass:
   ```bash
   go test ./...
   go vet ./...
   ```
4. Write clear, concise commit messages.
5. Open a PR against `main`.

### Pull Request Guidelines

- Keep PRs focused on a single change.
- Add tests for new functionality.
- Update documentation if behavior changes.
- Ensure CI passes before requesting review.

## Coding Standards

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions small and focused.
- Use meaningful variable and function names.
- Add comments for non-obvious logic.
- No CGO dependencies (builds must use `CGO_ENABLED=0`).

## Release Process

Releases are automated via GitHub Actions. When changes are pushed to `main`:
1. The CI workflow runs tests and vet.
2. The release workflow bumps the patch version, builds 4 platform binaries (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64), and creates a GitHub release.

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
