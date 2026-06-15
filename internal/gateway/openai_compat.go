package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// OpenAICompatRequest represents a request to /v1/chat/completions.
type OpenAICompatRequest struct {
	Model    string            `json:"model"`
	Messages []OpenAIMessage   `json:"messages"`
	Stream   bool              `json:"stream,omitempty"`
	Tools    []OpenAIToolDef   `json:"tools,omitempty"`
	Options  *OpenAICompatOpts `json:"options,omitempty"`
}

type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type OpenAIToolDef struct {
	Type     string         `json:"type"`
	Function OpenAIToolFunc `json:"function"`
}

type OpenAIToolFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAICompatOpts struct {
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// OpenAICompatResponse represents a response from /v1/chat/completions.
type OpenAICompatResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAICompatChoice `json:"choices"`
	Usage   *OpenAICompatUsage   `json:"usage,omitempty"`
}

type OpenAICompatChoice struct {
	Index        int            `json:"index"`
	Message      *OpenAIMessage `json:"message,omitempty"`
	Delta        *OpenAIMessage `json:"delta,omitempty"`
	FinishReason *string        `json:"finish_reason"`
}

type OpenAICompatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (s *Server) registerOpenAICompatRoutes() {
	s.mux.HandleFunc("POST /v1/chat/completions", s.handleOpenAICompatChat)
	s.mux.HandleFunc("GET /v1/models", s.handleOpenAICompatModels)
}

func (s *Server) handleOpenAICompatModels(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"object": "list",
		"data": []map[string]interface{}{
			{
				"id":       "golem-default",
				"object":   "model",
				"created":  time.Now().Unix(),
				"owned_by": "golem",
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOpenAICompatChat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("failed to read request body", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	defer r.Body.Close()

	var req OpenAICompatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.logger.Error("failed to parse JSON", slog.Any("error", err))
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "messages is required"})
		return
	}

	// Extract the last user message content
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" && req.Messages[i].Content != "" {
			userMessage = req.Messages[i].Content
			break
		}
	}
	if userMessage == "" {
		userMessage = req.Messages[len(req.Messages)-1].Content
	}

	// Build session ID from request (use model as session prefix for multi-tenant)
	sessionID := fmt.Sprintf("openai-compat-%s", req.Model)

	if req.Stream {
		s.handleOpenAICompatStream(w, r, sessionID, userMessage, req.Model)
		return
	}

	response, err := s.agent.HandleMessage(r.Context(), sessionID, userMessage)
	if err != nil {
		s.logger.Error("agent error", slog.Any("error", err), slog.String("model", req.Model))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := OpenAICompatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []OpenAICompatChoice{
			{
				Index: 0,
				Message: &OpenAIMessage{
					Role:    "assistant",
					Content: response,
				},
				FinishReason: strPtr("stop"),
			},
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOpenAICompatStream(w http.ResponseWriter, r *http.Request, sessionID, userMessage, model string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	streamer, isStreaming := s.agent.(StreamingAgentHandler)
	if !isStreaming {
		response, err := s.agent.HandleMessage(r.Context(), sessionID, userMessage)
		if err != nil {
			s.logger.Error("agent error", slog.Any("error", err))
			fmt.Fprintf(w, "data: {\"error\":\"internal server error\"}\n\n")
			flusher.Flush()
			return
		}
		chunk := OpenAICompatResponse{
			ID:      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []OpenAICompatChoice{
				{
					Index: 0,
					Delta: &OpenAIMessage{
						Role:    "assistant",
						Content: response,
					},
					FinishReason: strPtr("stop"),
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	tokens := make(chan string, 32)
	errCh := make(chan error, 1)

	go func() {
		errCh <- streamer.HandleMessageStream(r.Context(), sessionID, userMessage, tokens)
	}()

	chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())

	// Send role chunk first
	roleChunk := OpenAICompatResponse{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []OpenAICompatChoice{
			{
				Index: 0,
				Delta: &OpenAIMessage{
					Role: "assistant",
				},
			},
		},
	}
	data, _ := json.Marshal(roleChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	for token := range tokens {
		chunk := OpenAICompatResponse{
			ID:      chunkID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   model,
			Choices: []OpenAICompatChoice{
				{
					Index: 0,
					Delta: &OpenAIMessage{
						Content: token,
					},
				},
			},
		}
		data, _ := json.Marshal(chunk)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	if err := <-errCh; err != nil {
		s.logger.Error("stream error", slog.Any("error", err))
	}

	// Send finish chunk
	finishChunk := OpenAICompatResponse{
		ID:      chunkID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []OpenAICompatChoice{
			{
				Index:        0,
				FinishReason: strPtr("stop"),
			},
		},
	}
	data, _ = json.Marshal(finishChunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func strPtr(s string) *string {
	return &s
}
