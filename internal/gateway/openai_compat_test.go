package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strings77wzq/golem/foundation/logger"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "Hello from agent"}
	cfg := DefaultServerConfig()
	return NewServer(cfg, agent, log)
}

func TestOpenAICompatModels(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest("GET", "/v1/models", nil)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["object"] != "list" {
		t.Errorf("expected object 'list', got %v", resp["object"])
	}
}

func TestOpenAICompatChat(t *testing.T) {
	server := newTestServer(t)
	body := OpenAICompatRequest{
		Model: "golem-default",
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp OpenAICompatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Object != "chat.completion" {
		t.Errorf("expected object 'chat.completion', got %q", resp.Object)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice")
	}
	if resp.Choices[0].Message == nil {
		t.Fatal("expected message in choice")
	}
	if resp.Choices[0].Message.Content != "Hello from agent" {
		t.Errorf("expected 'Hello from agent', got %q", resp.Choices[0].Message.Content)
	}
}

func TestOpenAICompatChatEmptyMessages(t *testing.T) {
	server := newTestServer(t)
	body := `{"model":"golem-default","messages":[]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOpenAICompatChatInvalidJSON(t *testing.T) {
	server := newTestServer(t)
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestOpenAICompatStream(t *testing.T) {
	server := newTestServer(t)
	body := OpenAICompatRequest{
		Model:  "golem-default",
		Stream: true,
		Messages: []OpenAIMessage{
			{Role: "user", Content: "Hello"},
		},
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected content-type 'text/event-stream', got %q", contentType)
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("[DONE]")) {
		t.Error("expected [DONE] in stream")
	}
}
