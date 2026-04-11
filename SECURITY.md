# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.2.x   | :white_check_mark: |
| < 1.2   | :x:                |

We only provide security fixes for the latest minor release.

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email **security@shellrelay.com** with:

1. A description of the vulnerability.
2. Steps to reproduce (if applicable).
3. The potential impact.
4. Any suggested fixes (optional).

## Response Timeline

- **Acknowledgment**: Within 48 hours.
- **Initial assessment**: Within 5 business days.
- **Fix and disclosure**: We aim to release a fix within 30 days of confirmation.

## Scope

The following are in scope:

- The ShellRelay runner binary (`shellrelay`)
- The Docker image and entrypoint
- The install script (`install.sh`)
- WebSocket connection security (token handling, TLS)
- PTY session isolation
- Config file credential storage

The following are **out of scope**:

- The ShellRelay relay server (report separately at security@shellrelay.com)
- The ShellRelay web UI
- Third-party dependencies (report to the upstream project)

## Disclosure Policy

We follow coordinated disclosure. We will:
1. Work with you to understand and validate the report.
2. Develop and test a fix.
3. Release the fix and publish a security advisory.
4. Credit you in the advisory (unless you prefer anonymity).

Thank you for helping keep ShellRelay secure.
