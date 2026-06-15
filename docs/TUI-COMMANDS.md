# TUI Commands Reference

Golem's interactive TUI supports slash commands for session management and tool control.

## Commands

| Command | Description |
|---------|-------------|
| `/compact` | Compress conversation history to save context window |
| `/clear` | Clear conversation history |
| `/fork <index> <message>` | Fork session from a message index |
| `/help` | Show available commands |
| `/quit` | Exit the application (also `q` key) |

## @file Reference

Type `@path/to/file` in the input to inject file content into the prompt.

```
User: 分析表结构 @schema.sql
LLM receives file content in <file> tags
```

- Supports relative paths, absolute paths, `~` expansion
- Files > 50KB are truncated
- Binary files are skipped (original `@path` preserved)
- Multiple `@` references in one message are supported

### Examples

```bash
# Single file
@schema.sql

# Multiple files
@a.txt and @b.txt

# Inline with text
分析这个配置 @config.yaml 是否有问题
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `q` | Quit (when input is empty) |
| `Ctrl+C` | Quit |
| `Up/Down` | Scroll viewport |
| `PgUp/PgDn` | Page scroll |
| `Home/End` | Jump to top/bottom |

## Slash Commands via Registry

New commands can be added by implementing the `SlashCommand` interface:

```go
type SlashCommand interface {
    Name() string
    Description() string
    Execute(m *Model) tea.Cmd
}
```

Register in `tui.go` `New()`:

```go
cmds.Register(myNewCmd{})
```
