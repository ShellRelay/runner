## Summary

Brief description of the changes.

## Motivation

Why is this change needed? Link to any related issues.

Fixes #

## Changes

- 
- 

## PR Title Format

PR titles **must** follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

**Scopes (optional):** `runner`, `config`, `relay`, `session`, `docker`, `install`, `upgrade`, `announce`, `daemon`, `deps`

**Examples:**
- `feat: add session timeout support`
- `fix(relay): handle reconnect on network change`
- `docs: update install instructions`
- `build(deps): bump golang.org/x/net to 0.38.0`

## Testing

- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] Tested manually on (OS/arch): 

## Checklist

- [ ] Code follows the project's coding standards
- [ ] No CGO dependencies added
- [ ] Documentation updated (if applicable)
- [ ] No secrets or credentials in the diff
