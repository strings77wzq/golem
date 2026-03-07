# Beginner Labs

Three hands-on labs to learn Golem by doing. Each lab is self-contained and runnable in 5 minutes.

**Prerequisites**: Go 1.25+, git, a terminal.

---

## Lab A: Mock Provider One-Shot

Goal: Run a complete agent request using the built-in mock provider (no API key needed).

### Steps

1. **Clone and build**

```bash
git clone https://github.com/strings77wzq/golem.git
cd golem
CGO_ENABLED=0 go build -o build/golem ./cmd/golem
```

2. **Create config with mock provider**

```bash
mkdir -p ~/.golem
cat > ~/.golem/config.json << 'EOF'
{
  "agents": {
    "defaults": {
      "model_name": "mock",
      "max_tokens": 1024,
      "system_prompt": "You are a helpful assistant."
    }
  },
  "model_list": [
    {
      "model_name": "mock",
      "model": "mock/echo",
      "api_key": ""
    }
  ]
}
EOF
```

3. **Run one-shot query**

```bash
./build/golem agent -m "What is 2 + 2?"
```

**Expected output**: The mock provider echoes back a simple response. The agent completes in <1 second.

### What just happened?

1. `agent -m` triggers one-shot mode (no TUI)
2. Agent loads config, selects `"mock"` provider from `model_list`
3. ReAct loop runs: think → act (no tools) → respond
4. Response printed to stdout

---

## Lab B: TUI Streaming

Goal: Experience token-by-token streaming in the Bubble Tea TUI.

### Steps

1. **Start TUI** (must be on a TTY, not piped)

```bash
./build/golem agent
```

2. **In the TUI**, type:

```
Hello, what can you do?
```

3. **Watch tokens stream in** — you'll see characters appear one by one.

4. **Exit**: press `Ctrl+C` or type `/quit`

### What just happened?

1. TTY detection → Bubble Tea TUI auto-activates
2. `HandleMessageStream` → tokens flow through channel
3. `waitNextToken` recursive Cmd pattern renders each token
4. SSE not used here — it's local TUI streaming via Go channels

### Troubleshooting

| Problem | Solution |
|---------|----------|
| TUI doesn't start | Ensure stdout is a TTY: `golem agent` (not `echo "x" | golem agent`) |
| No streaming | Mock provider doesn't support streaming; use OpenAI/Anthropic with real API key |
| Mouse issues | Normal — mouse disabled by default for Termux compatibility |

---

## Lab C: Custom Echo Tool

Goal: Build and register a custom tool, then call it via the agent.

### Steps

1. **Create a simple tool** (`core/tools/echo/echo.go`)

```go
package echo

import (
	"context"
	"fmt"

	"github.com/strings77wzq/golem/core/tools"
)

// Echo tool repeats the input back with a prefix
type Echo struct{}

func New() *Echo { return &Echo{} }

func (e *Echo) Name() string        { return "echo" }
func (e *Echo) Description() string { return "Echoes back the input text with a prefix." }

func (e *Echo) Parameters() []tools.ToolParameter {
	return []tools.ToolParameter{
		{Name: "text", Type: "string", Required: true, Description: "Text to echo back"},
	}
}

func (e *Echo) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	text, ok := args["text"].(string)
	if !ok {
		return &tools.ToolResult{IsError: true}, fmt.Errorf("text is required")
	}
	return &tools.ToolResult{
		ForLLM:  fmt.Sprintf("Echoed: %s", text),
		ForUser: fmt.Sprintf("🔁 Echoed: %s", text),
	}, nil
}
```

2. **Register in main.go** (`cmd/golem/main.go`)

Find `buildToolRegistry()` and add:

```go
echoTool := echo.New()
if err := registry.Register(echoTool); err != nil {
    // handle error
}
```

3. **Rebuild**

```bash
CGO_ENABLED=0 go build -o build/golem ./cmd/golem
```

4. **Run and ask**

```bash
./build/golem agent -m "Please echo 'hello world'"
```

**Expected output**: The agent calls the `echo` tool, which returns "🔁 Echoed: hello world".

### What just happened?

1. Tool registered in alphabetical order (important for KV cache)
2. Agent sends tool definitions to LLM
3. LLM decides to call `echo` tool
4. Agent executes `echo.Execute()`, gets ToolResult
5. Both `ForLLM` (context for next LLM turn) and `ForUser` (display) are emitted

---

## Next Steps

- **Add a real provider**: Edit `~/.golem/config.json`, add OpenAI API key, switch `model_name` to `"gpt4"`
- **Try streaming**: Use a real provider (not mock) to see SSE token streaming
- **Explore the code**:
  - [Agent ReAct Loop](../docs/study/02-agent-react-loop.md)
  - [Tool System](../docs/study/03-tool-system.md)
  - [TUI Channel](../docs/study/07-tui-channel.md)

---

## Common Issues

| Error | Cause | Fix |
|-------|-------|-----|
| `provider not found` | `model_name` doesn't match any entry in `model_list` | Check config.json spelling |
| `API key missing` | `${OPENAI_API_KEY}` not set in environment | `export OPENAI_API_KEY="sk-..."` |
| `config.json not found` | Default location is `~/.golem/config.json` | Use `--config` flag or run `golem init` |
