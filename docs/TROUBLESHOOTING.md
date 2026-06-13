# Troubleshooting (First-Run Focus)

Use this ladder from top to bottom.

## 1) `golem: command not found`

### Cause

Binary path is not in current shell PATH.

### Fix

```bash
export PATH="$HOME/go/bin:$PATH"
which golem
golem version
```

If fixed, persist it in your shell rc file.

## 2) `golem init` fails or exits unexpectedly

### Cause

Config directory/file permission issue or interrupted setup.

### Fix

```bash
mkdir -p ~/.golem
golem init
```

Then verify:

```bash
golem status
```

## 3) Agent returns provider/auth errors

### Cause

Invalid API key, wrong model/provider mapping, or blocked network.

### Fix

```bash
golem status
golem config list
```

Re-run init to rewrite provider config if needed:

```bash
golem init
```

## 4) TUI does not appear

### Cause

Current session is non-TTY (pipe/redirect).

### Fix

```bash
golem agent
```

Do not pipe input for TUI mode.

## 5) Termux specific issues

### Cause

Missing Go package, PATH not set, or terminal environment mismatch.

### Fix

```bash
pkg install golang git
go install github.com/strings77wzq/golem/cmd/golem@latest
export PATH="$HOME/go/bin:$PATH"
golem version
```

## Escalation checklist

Before opening an issue, collect:

```bash
golem version
golem status
go env GOOS GOARCH
```

Attach command, full error output, and platform info.
