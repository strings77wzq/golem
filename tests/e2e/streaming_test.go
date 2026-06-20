package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/strings77wzq/golem/tests/e2e/helpers"
)

func TestGateway_StreamEmitsTokens(t *testing.T) {
	helpers.SkipIfUnavailable(t, "http://127.0.0.1:11434", "qwen3:0.5b")

	binPath := helpers.BuildGolem(t)
	configPath := writeOllamaConfig(t)

	// Find a free port
	port := freePort(t)

	// Start gateway
	cmd := exec.Command(binPath, "gateway",
		"-c", configPath,
		"--addr", fmt.Sprintf(":%d", port),
	)
	cmd.Dir = filepath.Join(helpers.RepoRoot(t))

	if err := cmd.Start(); err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for gateway to be ready
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	ready := waitForGateway(t, baseURL, 10*time.Second)
	if !ready {
		t.Fatal("gateway did not become ready")
	}

	// Send streaming request
	reqBody, _ := json.Marshal(map[string]string{
		"message": "Say hello in one word",
	})
	resp, err := http.Post(baseURL+"/api/chat/stream", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /api/chat/stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}

	// Parse SSE events
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	events := parseSSE(string(data))

	// Count distinct data events (excluding [DONE])
	dataEvents := 0
	hasDone := false
	for _, evt := range events {
		if evt.data == "[DONE]" || evt.event == "done" {
			hasDone = true
		} else if evt.data != "" && evt.event != "error" {
			dataEvents++
		}
	}

	if dataEvents < 2 {
		t.Errorf("expected ≥ 2 distinct data events, got %d (events: %+v)", dataEvents, events)
	}
	if !hasDone {
		t.Error("expected 'done' terminator event")
	}
}

// sseEvent represents a parsed SSE event.
type sseEvent struct {
	event string
	data  string
}

// parseSSE parses SSE text into events.
func parseSSE(text string) []sseEvent {
	var events []sseEvent
	var current sseEvent
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "event: ") {
			current.event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current.data = strings.TrimPrefix(line, "data: ")
		} else if line == "" && (current.event != "" || current.data != "") {
			events = append(events, current)
			current = sseEvent{}
		}
	}
	if current.event != "" || current.data != "" {
		events = append(events, current)
	}
	return events
}

// freePort returns an available TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// waitForGateway polls the gateway until it responds or timeout.
func waitForGateway(t *testing.T, baseURL string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
