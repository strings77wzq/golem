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

## Lab D: RAG 本地知识库问答
**Goal**: 基于本地文档构建知识库，让Agent可以回答和文档相关的问题。

### 前置准备
在当前目录下创建`docs/`文件夹，添加几个测试文档：
```bash
mkdir -p docs
echo "# Golem 架构说明
Golem采用四层六边形架构：core、foundation、feature、internal。
core层不能依赖其他上层模块，所有接线都在cmd/golem/目录完成。" > docs/architecture.md

echo "# Golem 命令说明
golem agent: 启动Agent交互
golem init: 初始化配置
golem gateway: 启动HTTP网关
支持的flags: --rag, --mcp, --skills" > docs/commands.md
```

### 步骤
1. **启动Agent并开启RAG**
```bash
./build/golem agent --rag ./docs
```

2. **提问关于文档的问题**
```
Golem的架构有几层？分别是什么？
```
**Expected output**: Agent会调用`rag_retrieve`工具检索相关文档内容，然后给出准确回答。

3. **再提问命令相关问题**
```
golem有哪些主要命令？
```
**Expected output**: Agent会返回文档中列出的三个主要命令和支持的flags。

### 原理说明
1. `--rag ./docs` 会自动索引目录下所有`.md`/`.txt`文件
2. 索引过程会计算文本向量，存储在内存中
3. 用户提问时，Agent自动调用`rag_retrieve`工具检索最相关的文档片段
4. 检索结果会被注入到LLM上下文中，让回答基于文档内容

---

## Lab E: MCP 外部工具集成
**Goal**: 通过MCP协议接入外部工具，扩展Agent能力。我们将使用一个简单的MCP演示服务器。

### 前置准备
1. 安装Python 3.8+
2. 下载一个简单的MCP演示服务器：
```bash
# 创建一个简单的MCP echo服务器
cat > mcp_echo_server.py << 'EOF'
import sys
import json

def main():
    for line in sys.stdin:
        try:
            req = json.loads(line.strip())
            if req.get("method") == "tools/list":
                resp = {
                    "jsonrpc": "2.0",
                    "id": req["id"],
                    "result": {
                        "tools": [
                            {
                                "name": "echo",
                                "description": "Echo back input text",
                                "parameters": {
                                    "type": "object",
                                    "properties": {
                                        "text": {"type": "string", "description": "Text to echo"}
                                    },
                                    "required": ["text"]
                                }
                            }
                        ]
                    }
                }
                print(json.dumps(resp))
                sys.stdout.flush()
            elif req.get("method") == "tools/call":
                tool_name = req["params"]["name"]
                args = req["params"]["arguments"]
                if tool_name == "echo":
                    resp = {
                        "jsonrpc": "2.0",
                        "id": req["id"],
                        "result": {
                            "content": [{"type": "text", "text": f"MCP Echo: {args['text']}"}]
                        }
                    }
                    print(json.dumps(resp))
                    sys.stdout.flush()
        except Exception as e:
            pass

if __name__ == "__main__":
    main()
EOF
```

### 步骤
1. **启动Agent并加载MCP服务器**
```bash
./build/golem agent --mcp '[{"command": "python", "args": ["mcp_echo_server.py"]}]'
```

2. **让Agent调用MCP工具**
```
请使用echo工具返回"hello mcp"
```
**Expected output**: Agent会调用`mcp_echo`工具，返回"MCP Echo: hello mcp"。

### 原理说明
1. `--mcp` flag 会启动配置的MCP服务器进程
2. Agent通过STDIO和MCP服务器通信，遵循JSON-RPC 2.0协议
3. Agent自动从MCP服务器获取工具列表，注册到全局工具注册表
4. 所有MCP工具自动添加`mcp_`前缀，避免和内置工具重名

---

## Lab F: 内置技能使用
**Goal**: 使用Golem内置的技能，快速获得特定场景的能力。

### 步骤
1. **启动Agent并启用技能**
```bash
./build/golem agent --skills summarize,code-review
```

2. **使用summarize技能总结长文本**
```
请总结以下内容：
Golem是一个纯Go实现的AI Agent框架，支持7大LLM厂商，具备MCP、RAG、技能系统等能力，
可以在Linux、macOS、Windows以及Android Termux上运行，采用四层六边形架构，
测试覆盖率79.2%，支持Docker/K8s云原生部署。
```
**Expected output**: Agent会调用`summarize`技能，返回简洁的摘要。

3. **使用code-review技能评审代码**
```
请评审这段Go代码：
func add(a, b int) int {
    return a + b
}
```
**Expected output**: Agent会调用`code-review`技能，给出代码评审意见。

### 原理说明
1. `--skills` flag 加载指定的内置技能
2. 每个技能都是一个预定义的工作流，暴露为LLM可调用的工具
3. 技能封装了特定领域的prompt和处理逻辑，不需要用户编写复杂提示词
4. 技能执行结果也遵循ToolResult格式，分为ForLLM和ForUser两部分

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
