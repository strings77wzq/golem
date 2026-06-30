# Logging Architecture

Golem uses structured logging with `log/slog` (Go stdlib). All logs are JSON or text formatted, with automatic `trace_id` injection and component identification.

## Core Concepts

### Trace ID Propagation

Every request generates a unique `trace_id` (format: `trace-{16hex}`) that propagates across all components:

```
gateway → agent → tool → provider
```

Filter logs by trace ID:
```bash
# JSON format
cat logs.json | jq 'select(.trace_id == "trace-a1b2c3d4e5f67890")'

# Text format
grep "trace_id=trace-a1b2c3d4e5f67890" logs.txt
```

### Component Tagging

Each log entry includes a `component` field identifying the source module:

| Component | Source |
|-----------|--------|
| `agent` | Core ReAct loop, tool execution |
| `gateway` | HTTP server, OpenAI-compatible API |
| `mcp` | MCP protocol server |
| `rag` | Hybrid search (BM25 + vector) |
| `routing` | Provider fallback |
| `health` | Provider health checks |
| `security` | Auth, rate limiting, sandbox |
| `tui` | Terminal UI |
| `cli` | Command-line interface |
| `telegram` | Telegram bot adapter |

Filter by component:
```bash
# Show only agent logs
cat logs.json | jq 'select(.component == "agent")'

# Show gateway and security logs
cat logs.json | jq 'select(.component == "gateway" or .component == "security")'
```

## Log Levels

| Level | When to Use |
|-------|-------------|
| `DEBUG` | Detailed debugging info (disabled by default) |
| `INFO` | Normal operations (tool executed, request processed) |
| `WARN` | Degraded but recoverable (embedding failed, falling back to BM25) |
| `ERROR` | Failures requiring attention (provider error, security violation) |

## Field Templates

Predefined logging functions ensure consistent format:

### LogToolCall

```go
logger.LogToolCall(log, "sql_query", 123*time.Millisecond, nil)
// Output: INFO tool executed tool=sql_query duration_ms=123

logger.LogToolCall(log, "sql_query", 50*time.Millisecond, errors.New("table not found"))
// Output: ERROR tool execution failed tool=sql_query duration_ms=50 error="table not found"
```

### LogHTTPRequest

```go
logger.LogHTTPRequest(log, "POST", "/api/chat", 200, 120*time.Millisecond)
// Output: INFO http request method=POST path=/api/chat status=200 duration_ms=120

logger.LogHTTPRequest(log, "POST", "/api/chat", 429, 10*time.Millisecond)
// Output: WARN http request method=POST path=/api/chat status=429 duration_ms=10
```

### LogError

```go
logger.LogError(log, err, "database connection failed", "host", "localhost")
// Output: ERROR database connection failed error="connection refused" host=localhost
```

## Configuration

### Log Level

```bash
# Default: INFO
golem agent --db ./myapp.db

# Debug level
golem agent --db ./myapp.db --log-level debug

# Error only
golem agent --db ./myapp.db --log-level error
```

### Log Format

```bash
# Text (default, human-readable)
golem agent --db ./myapp.db --log-format text

# JSON (for log aggregation systems)
golem agent --db ./myapp.db --log-format json
```

## Integration with Log Aggregation

### Loki + Promtail

```yaml
# promtail-config.yaml
scrape_configs:
  - job_name: golem
    static_configs:
      - targets: [localhost]
        labels:
          job: golem
          __path__: /var/log/golem/*.log
    pipeline_stages:
      - json:
          expressions:
            trace_id: trace_id
            component: component
            level: level
```

### ELK Stack

Use JSON format and Filebeat to ship logs to Elasticsearch. The structured fields (`trace_id`, `component`, `tool`, `duration_ms`) map directly to Elasticsearch fields.

## Development Guidelines

### Do

- Use `LogToolCall` for all tool executions
- Use `LogHTTPRequest` for all HTTP handlers
- Use `LogError` for error logging with context
- Include `trace_id` in error messages for correlation

### Don't

- Use `fmt.Print` for logging (use logger instead)
- Log sensitive data (API keys, passwords, tokens)
- Create log entries without component identification
- Use unstructured log messages
