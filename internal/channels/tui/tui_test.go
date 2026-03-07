package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type mockHandler struct {
	tokens []string
	err    error
}

func (m *mockHandler) HandleMessageStream(_ context.Context, _ string, _ string, out chan<- string) error {
	defer close(out)
	for _, t := range m.tokens {
		out <- t
	}
	return m.err
}

func TestModelInitialState(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	if m.thinking {
		t.Error("expected thinking=false on init")
	}
	if len(m.messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(m.messages))
	}
	if m.input != "" {
		t.Errorf("expected empty input, got %q", m.input)
	}
}

func TestModelKeyInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = raw.(Model)
	raw, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = raw.(Model)

	if m.input != "hi" {
		t.Errorf("input = %q, want %q", m.input, "hi")
	}
}

func TestModelBackspace(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	for _, ch := range "abc" {
		raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = raw.(Model)
	}
	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = raw.(Model)

	if m.input != "ab" {
		t.Errorf("input after backspace = %q, want %q", m.input, "ab")
	}
}

func TestModelEnterSendsMessage(t *testing.T) {
	handler := &mockHandler{tokens: []string{"hello"}}
	m := New(context.Background(), "sess", handler)

	for _, ch := range "hi" {
		raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = raw.(Model)
	}

	raw, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = raw.(Model)

	if !m.thinking {
		t.Error("expected thinking=true after Enter")
	}
	if len(m.messages) != 1 || m.messages[0].role != roleUser || m.messages[0].content != "hi" {
		t.Errorf("unexpected messages: %+v", m.messages)
	}
	if m.input != "" {
		t.Errorf("expected input cleared, got %q", m.input)
	}
}

func TestModelTokenReceived(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(tokenMsg("hello "))
	m = raw.(Model)
	raw, _ = m.Update(tokenMsg("world"))
	m = raw.(Model)

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(m.messages))
	}
	if m.messages[0].content != "hello world" {
		t.Errorf("content = %q, want %q", m.messages[0].content, "hello world")
	}
}

func TestModelDoneClears(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(doneMsg{})
	m = raw.(Model)

	if m.thinking {
		t.Error("expected thinking=false after doneMsg")
	}
}

func TestModelDoneWithError(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	raw, _ := m.Update(doneMsg{err: context.DeadlineExceeded})
	m = raw.(Model)

	if m.thinking {
		t.Error("expected thinking=false after doneMsg with error")
	}
	if m.lastError == "" {
		t.Error("expected lastError to be set")
	}
}

func TestModelQuitOnQ(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Error("expected quit command on 'q' with empty input")
	}
}

func TestModelNoQuitOnQWithInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "query"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		t.Error("should not quit when input is non-empty")
	}
}

func TestModelViewContainsInput(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.input = "test input"

	view := m.View()
	if !strings.Contains(view, "test input") {
		t.Errorf("view does not contain input: %q", view)
	}
}

func TestModelViewThinking(t *testing.T) {
	handler := &mockHandler{}
	m := New(context.Background(), "sess", handler)
	m.thinking = true

	view := m.View()
	if !strings.Contains(view, "…") {
		t.Errorf("view does not contain thinking indicator: %q", view)
	}
}
