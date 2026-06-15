# Built-in Tools Reference

Golem ships with the following built-in tools. Tools are registered in alphabetical order for LLM KV-cache optimization.

## Default Tools

| Tool | Description |
|------|-------------|
| `exec` | Execute shell commands in a sandboxed workspace (allowlist-based security) |
| `file_list` | List files in a directory |
| `file_read` | Read file contents |
| `file_write` | Write content to a file |
| `think` | Reasoning scratchpad for step-by-step thinking (no side effects) |
| `web_search` | Web search via DuckDuckGo |

## Database Tools (via `--db` flag)

| Tool | Description |
|------|-------------|
| `sql_query` | Execute SQL SELECT queries (read-only by default) |
| `sql_schema` | Get database schema information |
| `sql_analyze` | Analyze table data distribution |

## Infrastructure Tools (via `--infra` flag)

| Tool | Description |
|------|-------------|
| `kubectl` | Kubernetes operations |
| `docker` | Docker operations |
| `helm` | Helm chart operations |

## Tool Security Model

| Operation | Default | With Permission |
|-----------|---------|-----------------|
| SELECT queries | ✅ Allowed | ✅ |
| INSERT/UPDATE | ❌ Blocked | ✅ (requires WHERE) |
| DELETE | ❌ Blocked | ✅ (requires WHERE) |
| Shell commands | ⚠️ Allowlist only | ✅ |

## Adding Custom Tools

Implement the `tools.Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() []ToolParameter
    Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}
```

Register in `internal/wiring/tools.go`:

```go
func BuildToolRegistry(workspace string) *tools.Registry {
    registry := tools.NewRegistry()
    registry.Register(yourNewTool)
    // ...
    return registry
}
```

## Hook System

Tools can be intercepted by shell hooks:

```yaml
# agent.yaml
hooks:
  pre_tool_use:
    command: ./scripts/validate.sh    # receives JSON stdin, returns {"allowed":bool}
  post_tool_use:
    command: ./scripts/audit.sh       # receives JSON stdin with tool result
```

Hook input JSON:
```json
{
  "session_id": "abc123",
  "tool_name": "sql_query",
  "tool_input": {"sql": "SELECT * FROM users"}
}
```

Hook output JSON:
```json
{"allowed": true}
```
