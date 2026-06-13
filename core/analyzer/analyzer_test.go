package analyzer

import (
	"testing"
)

func TestAnalyzeSQLDuplicateEntry(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceSQL, "ERROR 1062 (23000): Duplicate entry '123' for key 'PRIMARY'")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
	if issues[0].Severity != SeverityMedium {
		t.Errorf("severity = %q, want medium", issues[0].Severity)
	}
}

func TestAnalyzeSQLLockTimeout(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceSQL, "ERROR 1205 (HY000): Lock wait timeout exceeded")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
	if issues[0].Severity != SeverityHigh {
		t.Errorf("severity = %q, want high", issues[0].Severity)
	}
}

func TestAnalyzeSQLTooManyConnections(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceSQL, "ERROR 1040 (08004): Too many connections")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
	if issues[0].Severity != SeverityCritical {
		t.Errorf("severity = %q, want critical", issues[0].Severity)
	}
}

func TestAnalyzeDockerPanic(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceDocker, "panic: runtime error: index out of range")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
	if issues[0].Severity != SeverityCritical {
		t.Errorf("severity = %q, want critical", issues[0].Severity)
	}
}

func TestAnalyzeDockerOOM(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceDocker, "out of memory")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestAnalyzeK8sCrashLoop(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceK8s, "Back-off restarting failed container CrashLoopBackOff")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestAnalyzeK8sOOMKilled(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceK8s, "OOMKilled")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestAnalyzeRedisOOM(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceRedis, "OOM command not allowed")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestAnalyzeLogsMultiple(t *testing.T) {
	a := NewAnalyzer()
	lines := []string{
		"ERROR: Duplicate entry",
		"ERROR: Duplicate entry",
		"OK: no problem",
	}
	issues := a.AnalyzeLogs(SourceSQL, lines)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue type, got %d", len(issues))
	}
	if issues[0].Count != 2 {
		t.Errorf("count = %d, want 2", issues[0].Count)
	}
}

func TestAnalyzeCleanLog(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceSQL, "Query completed successfully in 0.05s")
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestAnalyzeUnknownSource(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog("unknown", "some error")
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for unknown source, got %d", len(issues))
	}
}

func TestFormatIssues(t *testing.T) {
	issues := []Issue{
		{Severity: SeverityCritical, Issue: "OOM", Count: 3, FixCommand: "increase memory"},
	}
	output := FormatIssues(issues)
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestFormatIssuesEmpty(t *testing.T) {
	output := FormatIssues(nil)
	if output != "No issues detected." {
		t.Errorf("unexpected output: %q", output)
	}
}

func TestIssueFixSuggestion(t *testing.T) {
	a := NewAnalyzer()
	issues := a.AnalyzeLog(SourceSQL, "Lock wait timeout exceeded")
	if len(issues) == 0 {
		t.Fatal("expected issues")
	}
	if issues[0].FixSQL == "" {
		t.Error("expected SQL fix suggestion")
	}
}
