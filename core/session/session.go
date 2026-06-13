// Package session manages conversation state for the AI agent. Each
// conversation is tracked as a [Session] containing a slice of messages.
// Sessions are persisted via the [SessionStore] interface, which has two
// implementations: [MemoryStore] (in-process, no disk) and [SQLiteAdapter]
// (persistent, uses modernc.org/sqlite with CGO_ENABLED=0).
package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/strings77wzq/golem/core/providers"
)

// ExportData represents a session export in JSON format.
type ExportData struct {
	Version    string        `json:"version"`
	ExportedAt time.Time     `json:"exported_at"`
	Session    ExportSession `json:"session"`
}

// ExportSession contains the session data for export.
type ExportSession struct {
	ID        string              `json:"id"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Messages  []providers.Message `json:"messages"`
}

// Session holds a conversation's state.
type Session struct {
	ID        string
	Messages  []providers.Message
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.RWMutex
}

// NewSession creates a new session with the given ID.
func NewSession(id string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		Messages:  []providers.Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage appends a message and updates timestamp.
func (s *Session) AddMessage(msg providers.Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

// GetMessages returns a copy of all messages (thread-safe).
func (s *Session) GetMessages() []providers.Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	messages := make([]providers.Message, len(s.Messages))
	copy(messages, s.Messages)
	return messages
}

// MessageCount returns the number of messages.
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Messages)
}

// Clear removes all messages.
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = []providers.Message{}
	s.UpdatedAt = time.Now()
}

// Export returns the session data in exportable JSON format.
func (s *Session) Export() *ExportData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &ExportData{
		Version:    "1.0",
		ExportedAt: time.Now().UTC(),
		Session: ExportSession{
			ID:        s.ID,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
			Messages:  s.Messages,
		},
	}
}

// ExportToJSON exports the session to JSON bytes.
func (s *Session) ExportToJSON() ([]byte, error) {
	data := s.Export()
	return json.MarshalIndent(data, "", "  ")
}

// ImportFromJSON imports session data from JSON bytes.
// Returns a new Session with the imported data.
func ImportFromJSON(jsonData []byte) (*Session, error) {
	var exportData ExportData
	if err := json.Unmarshal(jsonData, &exportData); err != nil {
		return nil, fmt.Errorf("parsing export data: %w", err)
	}

	if exportData.Version == "" {
		return nil, fmt.Errorf("missing version field")
	}

	if exportData.Session.ID == "" {
		return nil, fmt.Errorf("missing session ID")
	}

	s := &Session{
		ID:        exportData.Session.ID,
		Messages:  exportData.Session.Messages,
		CreatedAt: exportData.Session.CreatedAt,
		UpdatedAt: exportData.Session.UpdatedAt,
		mu:        sync.RWMutex{},
	}

	if s.Messages == nil {
		s.Messages = []providers.Message{}
	}

	return s, nil
}
