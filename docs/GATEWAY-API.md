# Gateway API Reference

HTTP API reference for Golem's built-in gateway server.

## Running the Gateway

```bash
# Default port 18790
golem gateway

# Custom port
golem gateway --addr :8080
```

## Base URL

```
http://localhost:18790
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/version` | API version |
| POST | `/api/chat` | Non-streaming chat |
| POST | `/api/chat/stream` | Streaming chat (SSE) |

---

## GET /health

Health check endpoint.

### Request

```bash
curl http://localhost:18790/health
```

### Response

```json
{
  "status": "ok",
  "timestamp": "2026-03-07T12:00:00Z"
}
```

---

## GET /api/version

Returns API version.

### Request

```bash
curl http://localhost:18790/api/version
```

### Response

```json
{
  "version": "dev"
}
```

---

## POST /api/chat

Non-streaming chat request.

### Request

```bash
curl -X POST http://localhost:18790/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id": "abc123", "message": "Hello"}'
```

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | No | Session ID for conversation continuity. Omit to start new session. |
| `message` | string | Yes | User message |

### Response

```json
{
  "session_id": "abc123",
  "response": "Hello! How can I help you today?"
}
```

### Error Response

```json
{
  "error": "message is required"
}
```

---

## POST /api/chat/stream

Streaming chat using Server-Sent Events (SSE).

### Request

```bash
curl -X POST http://localhost:18790/api/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"session_id": "abc123", "message": "Tell me a story"}'
```

### Response (SSE Stream)

```
data: Hello

data: !

data: Once

data:  upon

data:  a

data:  time

event: done
data: [DONE]
```

### SSE Format

| Event | Data | Description |
|-------|------|-------------|
| (default) | token string | Incremental token |
| `error` | error message | Stream error |
| `done` | `[DONE]` | Stream complete |

### Complete curl Example

```bash
curl -N -X POST http://localhost:18790/api/chat/stream \
  -H "Content-Type: application/json" \
  -d '{"message": "Count to 5"}' \
  2>/dev/null
```

Output:
```
data: 1

data: 2

data: 3

data: 4

data: 5

event: done
data: [DONE]
```

### Response Headers

| Header | Value |
|--------|-------|
| Content-Type | `text/event-stream` |
| Cache-Control | `no-cache` |
| Connection | `keep-alive` |
| X-Accel-Buffering | `no` |

---

## Authentication

Currently no authentication. For production, add:

- API key via `Authorization: Bearer <key>` header
- Rate limiting via `internal/security/ratelimit.go`

---

## Session Management

### Start New Session

Omit `session_id` or provide empty string:

```bash
curl -X POST http://localhost:18790/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

Response includes generated `session_id`:

```json
{
  "session_id": "sess_abc123",
  "response": "Hi!"
}
```

### Continue Session

Pass `session_id` from previous response:

```bash
curl -X POST http://localhost:18790/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id": "sess_abc123", "message": "What did I just say?"}'
```

---

## Error Codes

| HTTP Code | Error Message | Cause |
|-----------|--------------|-------|
| 400 | `invalid request body` | Malformed JSON |
| 400 | `invalid JSON` | JSON parse error |
| 400 | `message is required` | Empty message field |
| 500 | `internal server error` | Agent processing failed |
| 500 | `streaming not supported` | Agent doesn't implement streaming |

---

## JavaScript Example

```javascript
const response = await fetch('http://localhost:18790/api/chat/stream', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ message: 'Hello' })
});

const reader = response.body.getReader();
const decoder = new TextDecoder();

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  
  const text = decoder.decode(value);
  // Parse SSE: "data: token\n\n"
  for (const line of text.split('\n')) {
    if (line.startsWith('data: ')) {
      console.log(line.slice(6));
    }
  }
}
```

---

## Related

- [Gateway Server](../internal/gateway/server.go)
- [Streaming & Providers](../docs/study/06-streaming-and-providers.md)
- [TUI Channel](../docs/study/07-tui-channel.md)
