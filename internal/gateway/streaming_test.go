package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/foundation/logger"
)

// mockStreamingAgent implements StreamingAgentHandler.
type mockStreamingAgent struct {
	response  string
	err       error
	tokens    []string
	streamErr error
}

func (m *mockStreamingAgent) HandleMessage(ctx context.Context, sessionID, message string) (string, error) {
	return m.response, m.err
}

func (m *mockStreamingAgent) HandleMessageStream(ctx context.Context, sessionID, message string, tokens chan<- string) error {
	defer close(tokens)
	if m.streamErr != nil {
		return m.streamErr
	}
	for _, tok := range m.tokens {
		tokens <- tok
	}
	return nil
}

func TestOpenAICompatStream_NonStreamingAgent(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "hello world"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := `{"model":"test","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	// Call the stream handler directly
	server.handleOpenAICompatStream(rec, req, "test-session", "hi", "test-model")

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %s", ct)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, "data: ") {
		t.Error("expected SSE data frames")
	}
	if !strings.Contains(bodyStr, "data: [DONE]") {
		t.Error("expected [DONE] terminator")
	}
	if !strings.Contains(bodyStr, "hello world") {
		t.Error("expected response content in SSE stream")
	}
	if rec.flushCount == 0 {
		t.Error("expected Flush to be called")
	}
}

func TestOpenAICompatStream_StreamingAgent(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockStreamingAgent{
		response: "unused",
		tokens:   []string{"hello", " ", "world"},
	}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := `{"model":"test","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.handleOpenAICompatStream(rec, req, "test-session", "hi", "test-model")

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	bodyStr := rec.Body.String()

	// Should contain role chunk
	if !strings.Contains(bodyStr, `"role":"assistant"`) {
		t.Error("expected role chunk")
	}

	// Should contain all tokens
	for _, tok := range []string{"hello", " ", "world"} {
		if !strings.Contains(bodyStr, tok) {
			t.Errorf("expected token '%s' in stream", tok)
		}
	}

	// Should contain finish reason
	if !strings.Contains(bodyStr, `"finish_reason":"stop"`) {
		t.Error("expected finish chunk with stop reason")
	}

	if !strings.Contains(bodyStr, "data: [DONE]") {
		t.Error("expected [DONE] terminator")
	}

	// Should have flushed multiple times (role + tokens + finish + done)
	if rec.flushCount < 4 {
		t.Errorf("expected at least 4 flushes, got %d", rec.flushCount)
	}
}

func TestOpenAICompatStream_StreamingAgentError(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockStreamingAgent{
		streamErr: context.DeadlineExceeded,
	}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := `{"model":"test","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.handleOpenAICompatStream(rec, req, "test-session", "hi", "test-model")

	// Should still complete (stream error is logged, not returned as HTTP error)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOpenAICompatStream_NonStreamingAgentError(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "", err: context.DeadlineExceeded}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := `{"model":"test","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.handleOpenAICompatStream(rec, req, "test-session", "hi", "test-model")

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	bodyStr := rec.Body.String()
	if !strings.Contains(bodyStr, `"error"`) {
		t.Error("expected error in SSE stream")
	}
}

func TestOpenAICompatStream_SSEFormat(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockStreamingAgent{
		tokens: []string{"test"},
	}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := `{"model":"test","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := newFlushRecorder()

	server.handleOpenAICompatStream(rec, req, "test-session", "hi", "gpt-4")

	bodyStr := rec.Body.String()
	lines := strings.Split(bodyStr, "\n")

	// Each data line should start with "data: "
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			t.Errorf("expected SSE format 'data: ...', got: %s", line)
			continue
		}
		// Verify JSON is valid
		jsonStr := strings.TrimPrefix(line, "data: ")
		var resp OpenAICompatResponse
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			t.Errorf("invalid JSON in SSE frame: %v", err)
		}
		if resp.Model != "gpt-4" {
			t.Errorf("expected model 'gpt-4', got '%s'", resp.Model)
		}
	}
}
