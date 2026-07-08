package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/strings77wzq/golem/foundation/logger"
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

func (s *Server) handleOpenAICompatModels(w http.ResponseWriter, _ *http.Request) {
	data := make([]map[string]interface{}, 0, len(s.modelList))
	for _, m := range s.modelList {
		created := m.Created
		if created == 0 {
			created = time.Now().Unix()
		}
		data = append(data, map[string]interface{}{
			"id":       m.ID,
			"object":   "model",
			"created":  created,
			"owned_by": m.Vendor,
		})
	}
	// Fallback to default if no models configured
	if len(data) == 0 {
		data = append(data, map[string]interface{}{
			"id":       "golem-default",
			"object":   "model",
			"created":  time.Now().Unix(),
			"owned_by": "golem",
		})
	}
	resp := map[string]interface{}{
		"object": "list",
		"data":   data,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOpenAICompatChat(w http.ResponseWriter, r *http.Request) {
	// Inject trace_id for request correlation
	ctx := logger.WithTraceID(r.Context(), logger.NewTraceID())

	body, err := readLimitedBody(w, r)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
			return
		}
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

	// Each request gets a unique session to prevent cross-user data leakage
	// when multiple clients use the same model endpoint.
	sessionID := fmt.Sprintf("openai-compat-%s", uuid.New().String())

	if req.Stream {
		s.handleOpenAICompatStream(w, r, sessionID, userMessage, req.Model)
		return
	}

	response, err := s.agent.HandleMessage(ctx, sessionID, userMessage)
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
	// Inject trace_id for request correlation
	ctx := logger.WithTraceID(r.Context(), logger.NewTraceID())

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	// SSE endpoint: disable write timeout so long-running streams aren't cut
	// off by the server-level WriteTimeout.
	if rc := http.NewResponseController(w); rc != nil {
		_ = rc.SetWriteDeadline(time.Time{})
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	streamer, isStreaming := s.agent.(StreamingAgentHandler)
	if !isStreaming {
		response, err := s.agent.HandleMessage(ctx, sessionID, userMessage)
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
		errCh <- streamer.HandleMessageStream(ctx, sessionID, userMessage, tokens)
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

streamLoop:
	for {
		select {
		case token, ok := <-tokens:
			if !ok {
				if err := <-errCh; err != nil {
					s.logger.Error("stream error", slog.Any("error", err))
				}
				break streamLoop
			}
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
		case <-r.Context().Done():
			return
		}
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
