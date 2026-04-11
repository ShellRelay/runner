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
4. Open a PR against `main`.

### Commit Messages and PR Titles

This project follows [Conventional Commits](https://www.conventionalcommits.org/). **PR titles are enforced by CI** and must match this format:

```
<type>[optional scope]: <description>
```

The description must start with a lowercase letter.

#### Types

| Type | When to use |
|---|---|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation only |
| `style` | Formatting, missing semicolons, etc. (no code change) |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `build` | Build system or dependency changes |
| `ci` | CI/CD configuration changes |
| `chore` | Maintenance tasks (no production code change) |
| `revert` | Reverting a previous commit |

#### Scopes (optional)

`runner`, `config`, `relay`, `session`, `docker`, `install`, `upgrade`, `announce`, `daemon`, `deps`

#### Examples

```
feat: add session timeout support
fix(relay): handle reconnect on network change
docs: update install instructions
build(deps): bump golang.org/x/net to 0.38.0
refactor(config): simplify config file parsing
test(session): add asciicast writer edge cases
ci: add PR title lint workflow
```

#### Breaking Changes

Append `!` after the type/scope for breaking changes:

```
feat!: change default relay URL format
fix(config)!: rename SHELLRELAY_URL to SHELLRELAY_RELAY_URL
```

### Pull Request Guidelines

- **PR title must follow Conventional Commits** (CI will check this).
- Keep PRs focused on a single change.
- Add tests for new functionality.
- Update documentation if behavior changes.
- Ensure CI passes before requesting review.
- Since we use squash merge, the PR title becomes the commit message in `main`.

## Coding Standards

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions small and focused.
- Use meaningful variable and function names.
- Add comments for non-obvious logic.
- No CGO dependencies (builds must use `CGO_ENABLED=0`).

## Release Process

Releases are managed through the `VERSION` file:

1. Bump the version in `VERSION` as part of your PR.
2. Once merged to `main`, the release workflow automatically:
   - Runs tests
   - Builds 4 platform binaries (darwin/arm64, darwin/amd64, linux/arm64, linux/amd64)
   - Generates checksums
   - Creates a GitHub release with the tag from `VERSION`
3. If the tag already exists, the workflow skips (safe for code-only PRs without a version bump).

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
