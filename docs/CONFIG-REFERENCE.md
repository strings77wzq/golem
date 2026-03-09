# Configuration Reference

Complete reference for `~/.golem/config.json`.

## File Location

```
~/.golem/config.json
```

Hot reload: Send `SIGHUP` to the golem process to reload config without restart.

## Schema

```json
{
  "agents": {
    "defaults": {
      "model_name": "gpt4",
      "max_tokens": 8192,
      "system_prompt": "You are Golem, a helpful AI assistant."
    }
  },
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-4o",
      "api_key": "${OPENAI_API_KEY}",
      "api_base": "https://api.openai.com/v1"
    }
  ]
}
```

## Full Field Reference

### `agents` (object)

Agent-level configuration.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `defaults` | object | Yes | - | Default settings for all agents |

### `agents.defaults` (object)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model_name` | string | Yes | `"gpt4"` | Short name referencing an entry in `model_list`. Used by CLI `-M` flag. |
| `max_tokens` | integer | No | `8192` | Maximum tokens in LLM response |
| `system_prompt` | string | No | `"You are Golem, a helpful AI assistant."` | System prompt injected at session start |

### `model_list` (array)

List of available models. Each entry:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `model_name` | string | Yes | - | Short alias (e.g., `"gpt4"`, `"claude"`, `"deepseek"`). Must be unique. Referenced by `agents.defaults.model_name`. |
| `model` | string | Yes | - | Full vendor/model ID in format `"vendor/model-id"`. Example: `"openai/gpt-4o"`, `"anthropic/claude-sonnet-4-20250514"` |
| `api_key` | string | Yes | - | API key. Supports `${ENV_VAR}` substitution: `"${OPENAI_API_KEY}"` reads from environment variable. |
| `api_base` | string | No | vendor default | Override default API endpoint. Example: `"https://api.openai.com/v1"` |

## Vendor/Model Format

The `model` field uses `"vendor/model-id"` format:

| Vendor | Example `model` value | API Base (default) |
|--------|----------------------|-------------------|
| OpenAI | `"openai/gpt-4o"` | `https://api.openai.com/v1` |
| Anthropic | `"anthropic/claude-sonnet-4-20250514"` | `https://api.anthropic.com` |
| DeepSeek | `"deepseek/deepseek-chat"` | `https://api.deepseek.com/v1` |
| Kimi (Moonshot) | `"moonshot/moonshot-v1-8k"` | `https://api.moonshot.cn/v1` |
| GLM (Zhipu) | `"zhipu/glm-4"` | `https://open.bigmodel.cn/api/paas/v4` |
| MiniMax | `"minimax/MiniMax-Text-01"` | `https://api.minimax.chat/v1` |
| Qwen (DashScope) | `"dashscope/qwen-plus"` | `https://dashscope.aliyuncs.com/api/v1` |
| Mock | `"mock/echo"` | N/A (built-in test provider) |

## Environment Variable Substitution

Use `${ENV_VAR}` in `api_key` field:

```json
{
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-4o",
      "api_key": "${OPENAI_API_KEY}"
    }
  ]
}
```

Set before running golem:

```bash
export OPENAI_API_KEY="sk-..."
golem agent -m "hello"
```

Or export all at once:

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export DEEPSEEK_API_KEY="sk-..."
golem agent
```

## CLI Flag Override

Use CLI flags to override configuration at runtime:

```bash
# Override default model
golem agent -M deepseek -m "hello"

# Enable MCP (Model Context Protocol) integration
# Accepts JSON string or path to JSON config file
golem agent --mcp '[{"command": "python", "args": ["mcp-server.py"]}]'
golem agent --mcp ./mcp-config.json

# Enable RAG (Retrieval-Augmented Generation)
# Accepts directory path (indexes all text files) or path to JSON document list
golem agent --rag ./docs
golem agent --rag ./rag-documents.json

# Enable built-in skills (comma-separated list)
golem agent --skills summarize,code-review
```

## Feature Configuration

### MCP Configuration
MCP config is a list of server definitions:
```json
[
  {
    "command": "python",
    "args": ["/path/to/mcp-server.py"],
    "env": {
      "API_KEY": "${MCP_SERVER_API_KEY}"
    }
  }
]
```

### RAG Document Configuration
RAG document list JSON format:
```json
[
  {
    "id": "doc-1",
    "title": "Golem Architecture",
    "content": "Golem uses a hexagonal architecture with 4 layers..."
  },
  {
    "id": "doc-2",
    "title": "CLI Usage",
    "content": "Golem supports multiple CLI flags for feature toggles..."
  }
]
```

### Available Skills
| Skill Name | Description |
|------------|-------------|
| `summarize` | Summarizes long text content into concise points |
| `code-review` | Performs code review with best practice suggestions |

## Complete Example

```json
{
  "agents": {
    "defaults": {
      "model_name": "deepseek",
      "max_tokens": 4096,
      "system_prompt": "You are a coding assistant. Reply in Chinese."
    }
  },
  "model_list": [
    {
      "model_name": "gpt4",
      "model": "openai/gpt-4o",
      "api_key": "${OPENAI_API_KEY}"
    },
    {
      "model_name": "claude",
      "model": "anthropic/claude-sonnet-4-20250514",
      "api_key": "${ANTHROPIC_API_KEY}"
    },
    {
      "model_name": "deepseek",
      "model": "deepseek/deepseek-chat",
      "api_key": "${DEEPSEEK_API_KEY}"
    },
    {
      "model_name": "mock",
      "model": "mock/echo",
      "api_key": ""
    }
  ]
}
```

## Related

- [`golem init`](07-tui-channel.md) — interactive setup wizard
- [`golem config`](README.md#configuration-management) — CLI config management
- [Provider System](../docs/study/04-provider-system.md) — provider architecture
