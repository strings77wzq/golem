package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/strings77wzq/golem/core/session"
	"github.com/strings77wzq/golem/foundation/logger"
)

// mockSessionStore implements SessionStore for testing.
type mockSessionStore struct {
	sessions map[string]*session.Session
	saveErr  error
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{sessions: make(map[string]*session.Session)}
}

func (m *mockSessionStore) Get(id string) (*session.Session, bool) {
	s, ok := m.sessions[id]
	return s, ok
}

func (m *mockSessionStore) Save(s *session.Session) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.sessions[s.ID] = s
	return nil
}

func TestHandleSessionExport_NoStore(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)
	// No session store set

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test-id/export", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestHandleSessionExport_NotFound(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)
	server.SetSessionStore(newMockSessionStore())

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/nonexistent/export", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleSessionExport_Success(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	store := newMockSessionStore()
	sess := session.NewSession("test-session")
	store.sessions["test-session"] = sess
	server.SetSessionStore(store)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/test-session/export", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	if cd := rec.Header().Get("Content-Disposition"); cd != "attachment; filename=test-session.json" {
		t.Errorf("expected Content-Disposition for test-session, got %s", cd)
	}
}

func TestHandleSessionImport_NoStore(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	body := bytes.NewBufferString(`{"id":"imported-session","messages":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/import", body)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}

func TestHandleSessionImport_InvalidJSON(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)
	server.SetSessionStore(newMockSessionStore())

	body := bytes.NewBufferString(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/import", body)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleSessionImport_Success(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	store := newMockSessionStore()
	server.SetSessionStore(store)

	// Create a valid session export
	sess := session.NewSession("original-id")
	exportData := sess.Export()
	body, _ := json.Marshal(exportData)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "imported" {
		t.Errorf("expected status 'imported', got '%s'", resp["status"])
	}
}

func TestHandleSessionImport_SaveError(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	store := newMockSessionStore()
	store.saveErr = context.DeadlineExceeded
	server.SetSessionStore(store)

	sess := session.NewSession("test-id")
	exportData := sess.Export()
	body, _ := json.Marshal(exportData)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleMetrics(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct == "" {
		t.Error("expected Content-Type to be set")
	}
}

func TestSetSessionStore(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	store := newMockSessionStore()
	server.SetSessionStore(store)

	// Verify store is wired by testing export works
	sess := session.NewSession("wired-test")
	store.sessions["wired-test"] = sess

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/wired-test/export", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 after SetSessionStore, got %d", rec.Code)
	}
}

func TestMountHandler(t *testing.T) {
	log := logger.NopLogger()
	agent := &mockAgentHandler{response: "test"}
	cfg := DefaultServerConfig()
	server := NewServer(cfg, agent, log)

	customHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("custom")) //nolint:errcheck
	})
	server.MountHandler("GET /custom", customHandler)

	req := httptest.NewRequest(http.MethodGet, "/custom", nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "custom" {
		t.Errorf("expected 'custom', got '%s'", rec.Body.String())
	}
}
