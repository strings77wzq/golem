# Testing Guide

How to test Golem: unit tests, integration tests, race detection, coverage.

## Quick Commands

```bash
# All tests
go test ./...

# With race detector (recommended)
go test -race ./...

# With coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Specific package
go test ./core/agent/...

# Benchmarks
go test -bench=. -benchmem ./internal/gateway/...
```

## Test Types

### Unit Tests

Most tests are unit tests. They are:
- Fast (< 1 second)
- Self-contained (no external dependencies)
- Located in `*_test.go` files alongside source

Example:

```bash
go test ./core/bus/... -v
```

### Integration Tests

Some tests require external services (API keys, network). These are marked:

```go
func TestOpenAIProvider(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // requires OPENAI_API_KEY environment variable
}
```

Run integration tests:

```bash
# Skip short tests
go test -short ./...

# Run all (requires API keys)
go test ./...
```

### Example Tests (godoc)

Nine `Example_*` functions demonstrate API usage via godoc:

```bash
# View examples
go doc -examples core/tools
go doc -examples core/bus
go doc -examples core/session
```

## Race Detector

**Always run with race detector before committing:**

```bash
go test -race ./...
```

This detects data races — critical for a concurrent Go project.

If race detected, output includes:
```
WARNING: DATA RACE
Read at 0x00c000...
Write at 0x00c000...
```

## Coverage

Current coverage: **79.2%**

### Per-package Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View per-package coverage
go tool cover -func=coverage.out
```

### Improving Coverage

1. Identify low-coverage packages:
   ```bash
   go tool cover -func=coverage.out | grep -v "100.0%"
   ```

2. Add tests for uncovered edge cases

3. Target: maintain ≥79% coverage

## API Key Tests

Tests requiring API keys check for environment variables:

| Provider | Env Variable |
|----------|-------------|
| OpenAI | `OPENAI_API_KEY` |
| Anthropic | `ANTHROPIC_API_KEY` |
| DeepSeek | `DEEPSEEK_API_KEY` |
| Kimi | `MOONSHOT_API_KEY` |
| GLM | `ZHIPU_API_KEY` |
| MiniMax | `MINIMAX_API_KEY` |
| Qwen | `DASHSCOPE_API_KEY` |

Run without API keys (skips integration tests):

```bash
go test -short ./...
```

## Linting

Before committing:

```bash
go vet ./...
```

For additional linting (requires golangci-lint):

```bash
golangci-lint run
```

## Testing TUI

The TUI (`internal/channels/tui/`) can be tested without starting a terminal:

```bash
# Test Model.Update without TTY
go test ./internal/channels/tui/... -v
```

See [TUI Testing Strategy](../docs/study/07-tui-channel.md#testing-strategy) for details.

## Testing Providers

Mock providers for testing without API keys:

```go
import "github.com/strings77wzq/golem/core/providers"

// Create mock
mock := providers.NewMockProvider(func(ctx context.Context, msgs []providers.Message) *providers.LLMResponse {
    return &providers.LLMResponse{
        Content: "mock response",
    }
})
```

See `core/providers/mock_test.go` for examples.

## CI Checks

GitHub Actions CI runs:

1. `go mod download`
2. `go mod verify`
3. `go vet ./...`
4. `go test -race ./...` (Ubuntu + macOS)
5. `CGO_ENABLED=0 go build`
6. `docker build`

All must pass for merge.

## Best Practices

1. **Always use `-race`** in local testing
2. **Keep tests fast** — mock external calls
3. **Name test files `*_test.go`** in same package
4. **Use table-driven tests** for multiple cases:
   ```go
   func TestEcho(t *testing.T) {
       tests := []struct {
           input    string
           expected string
       }{
           {"hello", "🔁 Echoed: hello"},
           {"", "🔁 Echoed: "},
       }
       for _, tt := range tests {
           t.Run(tt.input, func(t *testing.T) {
               // test logic
           })
       }
   }
   ```
5. **Skip integration tests in short mode** with `testing.Short()`

## Related

- [Testing in Go](https://go.dev/blog/subtesting)
- [Race Detector](https://go.dev/doc/articles/race_detector)
- [Example Tests](https://go.dev/blog/examples)
