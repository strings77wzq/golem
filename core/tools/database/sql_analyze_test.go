package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/strings77wzq/golem/core/database"
)

// fakeDriver is a minimal SQLDriver that records Query/Execute/GetSchemaForTable
// calls so the sql_analyze identifier guard can assert what reached the driver.
type fakeDriver struct {
	queries     []string
	schemaCalls []string
	knownTables map[string]bool
	schemaErr   error // when set, GetSchemaForTable returns this error
}

func (f *fakeDriver) Name() string { return "fake" }

func (f *fakeDriver) Query(_ context.Context, sql string, _ ...interface{}) ([]database.Row, error) {
	f.queries = append(f.queries, sql)
	return []database.Row{{"cnt": int64(7)}}, nil
}

func (f *fakeDriver) Execute(_ context.Context, _ string, _ ...interface{}) (database.Result, error) {
	return database.Result{}, nil
}

func (f *fakeDriver) GetSchema(_ context.Context) (string, error) { return "", nil }

func (f *fakeDriver) GetSchemaForTable(_ context.Context, table string) (string, error) {
	f.schemaCalls = append(f.schemaCalls, table)
	if f.schemaErr != nil {
		return "", f.schemaErr
	}
	if f.knownTables[table] {
		return "schema for " + table, nil
	}
	return "", errors.New("table not found: " + table)
}

func (f *fakeDriver) Ping(_ context.Context) error { return nil }
func (f *fakeDriver) Close() error                 { return nil }

func newFakeRegistry(t *testing.T, f *fakeDriver) *database.Registry {
	t.Helper()
	reg := database.NewRegistry()
	if err := reg.RegisterSQL("test", f); err != nil {
		t.Fatal(err)
	}
	reg.SetDefault("test")
	return reg
}

func TestSQLAnalyzeToolRejectsInjectionPayload(t *testing.T) {
	f := &fakeDriver{knownTables: map[string]bool{"users": true}}
	reg := newFakeRegistry(t, f)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "users; DROP TABLE users--",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Errorf("expected IsError for injection payload, got result: %s", result.ForLLM)
	}
	if strings.Contains(result.ForLLM, "DROP TABLE") {
		t.Errorf("injection payload leaked into error message: %s", result.ForLLM)
	}
	if len(f.queries) != 0 {
		t.Errorf("expected no Query issued for rejected identifier, got %d: %v", len(f.queries), f.queries)
	}
	if len(f.schemaCalls) != 0 {
		t.Errorf("charset check should fail before schema probe; probed: %v", f.schemaCalls)
	}
}

func TestSQLAnalyzeToolRejectsTableNotInSchema(t *testing.T) {
	f := &fakeDriver{knownTables: map[string]bool{"users": true}} // "nonexistent" absent
	reg := newFakeRegistry(t, f)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "nonexistent",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected IsError for table not in schema")
	}
	if len(f.queries) != 0 {
		t.Errorf("expected no count query for out-of-schema table, got: %v", f.queries)
	}
}

func TestSQLAnalyzeToolSchemaProbeError(t *testing.T) {
	f := &fakeDriver{
		knownTables: map[string]bool{"products": true},
		schemaErr:   errors.New("connection lost"),
	}
	reg := newFakeRegistry(t, f)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "products",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected IsError when schema probe fails")
	}
	if len(f.queries) != 0 {
		t.Errorf("expected no count query when probe fails, got: %v", f.queries)
	}
}

func TestSQLAnalyzeToolValidTableAllowed(t *testing.T) {
	f := &fakeDriver{knownTables: map[string]bool{"products": true}}
	reg := newFakeRegistry(t, f)
	tool := NewSQLAnalyzeTool(reg)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"table": "products",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected valid table to be allowed, got error: %s", result.ForLLM)
	}
	if len(f.queries) != 1 {
		t.Errorf("expected exactly one count query for valid table, got %d: %v", len(f.queries), f.queries)
	}
}
