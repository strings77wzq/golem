# Platform Matrix & Known Limits

This matrix shows official support level and first-run caveats.

| Platform | Arch | Status | Install | Notes |
|---|---|---|---|---|
| Linux | amd64 | ✅ Official | `go install` / release binary | Primary production target |
| Linux | arm64 | ✅ Official | release binary / source build | Verify provider network connectivity |
| Android (Termux) | arm64 | ✅ Official | `go install` / source build | Add `$HOME/go/bin` into PATH |
| macOS | amd64/arm64 | ⚠️ Community-friendly | `go install` / release binary | Functionality is expected to work, less CI focus |
| Windows | amd64/arm64 | ⚠️ Community-friendly | release binary | Prefer one-shot mode first, then TUI |

## Linux amd64 first-run checks

```bash
which golem
golem version
golem status
golem agent -m "hello"
```

## Termux first-run checks

```bash
pkg install golang git
go install github.com/strings77wzq/golem/cmd/golem@latest
export PATH="$HOME/go/bin:$PATH"
golem version
golem init
golem agent -m "hello"
```

## Known limits

- Termux environments may need manual PATH setup for `$HOME/go/bin`.
- TUI requires a real TTY; piping input falls back to plain output mode.
- If provider API is blocked by network/proxy, use `golem status` + troubleshooting ladder before assuming a code issue.
