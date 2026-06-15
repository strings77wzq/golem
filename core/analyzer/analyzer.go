// Package analyzer provides log analysis, issue diagnosis, and fix suggestion
// capabilities for the AI agent. It parses database, Docker, Kubernetes, and
// application logs, identifies known error patterns, and generates actionable
// fix suggestions.
package analyzer

import (
	"fmt"
	"regexp"
	"strings"
)

// Severity represents the severity level of an issue.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// LogSource represents the source of a log.
type LogSource string

const (
	SourceSQL    LogSource = "sql"
	SourceDocker LogSource = "docker"
	SourceK8s    LogSource = "k8s"
	SourceRedis  LogSource = "redis"
	SourceApp    LogSource = "app"
)

// ErrorPattern defines a known error pattern to match against.
type ErrorPattern struct {
	Pattern    string
	Regex      *regexp.Regexp
	Severity   Severity
	Issue      string
	FixSQL     string // SQL fix
	FixCommand string // Shell command fix
	FixYAML    string // YAML fix
	RiskLevel  string // "low", "medium", "high"
}

// Issue represents a detected problem in logs.
type Issue struct {
	Pattern    ErrorPattern
	Count      int
	FirstLine  string
	Severity   Severity
	Issue      string
	Suggestion string
	FixSQL     string
	FixCommand string
	FixYAML    string
}

// Analyzer analyzes logs and generates diagnoses.
type Analyzer struct {
	patterns map[LogSource][]ErrorPattern
}

// NewAnalyzer creates a new log analyzer with built-in patterns.
func NewAnalyzer() *Analyzer {
	a := &Analyzer{
		patterns: make(map[LogSource][]ErrorPattern),
	}
	a.registerBuiltinPatterns()
	return a
}

// registerBuiltinPatterns adds known error patterns for all sources.
func (a *Analyzer) registerBuiltinPatterns() {
	// SQL patterns
	a.patterns[SourceSQL] = []ErrorPattern{
		{Pattern: "Duplicate entry", Severity: SeverityMedium, Issue: "Primary key conflict", FixSQL: "-- Check for duplicate data before inserting", RiskLevel: "low"},
		{Pattern: "Lock wait timeout", Severity: SeverityHigh, Issue: "Transaction lock contention", FixSQL: "SHOW PROCESSLIST;\n-- Kill long-running transactions", RiskLevel: "medium"},
		{Pattern: "Too many connections", Severity: SeverityCritical, Issue: "Connection pool exhausted", FixSQL: "SET GLOBAL max_connections = 300;", RiskLevel: "high"},
		{Pattern: "Table.*doesn't exist", Severity: SeverityHigh, Issue: "Missing table", FixSQL: "-- Run migration to create the table", RiskLevel: "medium"},
		{Pattern: "Column.*doesn't exist", Severity: SeverityHigh, Issue: "Missing column", FixSQL: "-- ALTER TABLE ADD COLUMN ...", RiskLevel: "medium"},
		{Pattern: "Foreign key constraint", Severity: SeverityMedium, Issue: "FK violation", FixSQL: "-- Check referential integrity", RiskLevel: "low"},
		{Pattern: "Deadlock", Severity: SeverityHigh, Issue: "Transaction deadlock", FixSQL: "-- Reduce transaction scope", RiskLevel: "medium"},
		{Pattern: "Query was interrupted", Severity: SeverityMedium, Issue: "Query cancelled", FixSQL: "-- Optimize query or increase timeout", RiskLevel: "low"},
		{Pattern: "Out of memory", Severity: SeverityCritical, Issue: "OOM during query", FixSQL: "-- Increase max_heap_table_size or optimize query", RiskLevel: "high"},
		{Pattern: "Disk full", Severity: SeverityCritical, Issue: "Disk space exhausted", FixSQL: "-- Free disk space or add storage", RiskLevel: "high"},
	}

	// Docker patterns
	a.patterns[SourceDocker] = []ErrorPattern{
		{Pattern: "panic:", Severity: SeverityCritical, Issue: "Runtime panic", FixCommand: "Check stack trace, add input validation", RiskLevel: "medium"},
		{Pattern: "connection refused", Severity: SeverityHigh, Issue: "Service unreachable", FixCommand: "docker ps | grep <service>", RiskLevel: "medium"},
		{Pattern: "permission denied", Severity: SeverityHigh, Issue: "Permission error", FixCommand: "chmod +x or chown", RiskLevel: "medium"},
		{Pattern: "out of memory", Severity: SeverityCritical, Issue: "OOM killed", FixYAML: "resources:\n  limits:\n    memory: '256Mi'", RiskLevel: "high"},
		{Pattern: "no space left on device", Severity: SeverityCritical, Issue: "Disk full", FixCommand: "docker system prune -a", RiskLevel: "high"},
		{Pattern: "TLS handshake", Severity: SeverityMedium, Issue: "TLS/SSL error", FixCommand: "Check certificates or use --insecure", RiskLevel: "low"},
		{Pattern: "timeout", Severity: SeverityMedium, Issue: "Operation timeout", FixCommand: "Increase timeout or optimize slow operation", RiskLevel: "low"},
		{Pattern: "context canceled", Severity: SeverityLow, Issue: "Request cancelled", FixCommand: "Check if client disconnected", RiskLevel: "low"},
		{Pattern: "executable file not found", Severity: SeverityHigh, Issue: "Missing binary in image", FixYAML: "RUN chmod +x /app/binary", RiskLevel: "medium"},
		{Pattern: "address already in use", Severity: SeverityHigh, Issue: "Port conflict", FixCommand: "docker ps to check port usage", RiskLevel: "medium"},
	}

	// K8s patterns
	a.patterns[SourceK8s] = []ErrorPattern{
		{Pattern: "ImagePullBackOff", Severity: SeverityHigh, Issue: "Image pull failed", FixCommand: "kubectl describe pod <pod> --namespace=<ns>", RiskLevel: "medium"},
		{Pattern: "CrashLoopBackOff", Severity: SeverityCritical, Issue: "Container crashing", FixCommand: "kubectl logs <pod> --previous", RiskLevel: "high"},
		{Pattern: "OOMKilled", Severity: SeverityCritical, Issue: "Container OOM", FixYAML: "resources:\n  limits:\n    memory: '256Mi'", RiskLevel: "high"},
		{Pattern: "Evicted", Severity: SeverityHigh, Issue: "Pod evicted", FixYAML: "resources:\n  requests:\n    memory: '128Mi'", RiskLevel: "medium"},
		{Pattern: "Unhealthy", Severity: SeverityHigh, Issue: "Health check failed", FixYAML: "livenessProbe:\n  httpGet:\n    path: /health\n    port: 8080", RiskLevel: "medium"},
		{Pattern: "Insufficient", Severity: SeverityHigh, Issue: "Insufficient resources", FixCommand: "kubectl top nodes", RiskLevel: "medium"},
		{Pattern: "FailedScheduling", Severity: SeverityHigh, Issue: "Cannot schedule pod", FixCommand: "kubectl describe events --namespace=<ns>", RiskLevel: "medium"},
		{Pattern: "ErrImagePull", Severity: SeverityHigh, Issue: "Cannot pull image", FixCommand: "docker pull <image> to verify", RiskLevel: "medium"},
	}

	// Redis patterns
	a.patterns[SourceRedis] = []ErrorPattern{
		{Pattern: "NOAUTH", Severity: SeverityHigh, Issue: "Authentication required", FixCommand: "redis-cli AUTH <password>", RiskLevel: "medium"},
		{Pattern: "OOM command not allowed", Severity: SeverityCritical, Issue: "Redis memory full", FixCommand: "redis-cli MEMORY USAGE <key>", RiskLevel: "high"},
		{Pattern: "MISCONF", Severity: SeverityHigh, Issue: "Redis config error", FixCommand: "Check redis.conf or CONFIG SET", RiskLevel: "medium"},
		{Pattern: "LOADING", Severity: SeverityMedium, Issue: "Redis loading dataset", FixCommand: "Wait for RDB/AOF load to complete", RiskLevel: "low"},
	}
}

// AnalyzeLog analyzes a single log line and returns any matching issues.
func (a *Analyzer) AnalyzeLog(source LogSource, line string) []Issue {
	var issues []Issue

	patterns, ok := a.patterns[source]
	if !ok {
		return nil
	}

	for _, p := range patterns {
		if strings.Contains(line, p.Pattern) {
			issues = append(issues, Issue{
				Pattern:    p,
				Count:      1,
				FirstLine:  line,
				Severity:   p.Severity,
				Issue:      p.Issue,
				Suggestion: p.FixSQL,
				FixSQL:     p.FixSQL,
				FixCommand: p.FixCommand,
				FixYAML:    p.FixYAML,
			})
		}
	}

	return issues
}

// AnalyzeLogs analyzes multiple log lines and returns aggregated issues.
func (a *Analyzer) AnalyzeLogs(source LogSource, lines []string) []Issue {
	issueMap := make(map[string]*Issue)

	for _, line := range lines {
		issues := a.AnalyzeLog(source, line)
		for _, issue := range issues {
			key := issue.Issue
			if existing, ok := issueMap[key]; ok {
				existing.Count++
			} else {
				issueCopy := issue
				issueMap[key] = &issueCopy
			}
		}
	}

	var result []Issue
	for _, issue := range issueMap {
		result = append(result, *issue)
	}

	return result
}

// FormatIssues returns a human-readable string of detected issues.
func FormatIssues(issues []Issue) string {
	if len(issues) == 0 {
		return "No issues detected."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d issue(s):\n\n", len(issues)))

	for i, issue := range issues {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, strings.ToUpper(string(issue.Severity)), issue.Issue))
		sb.WriteString(fmt.Sprintf("   Count: %d occurrence(s)\n", issue.Count))
		if issue.FixSQL != "" {
			sb.WriteString(fmt.Sprintf("   SQL Fix: %s\n", issue.FixSQL))
		}
		if issue.FixCommand != "" {
			sb.WriteString(fmt.Sprintf("   Command: %s\n", issue.FixCommand))
		}
		if issue.FixYAML != "" {
			sb.WriteString(fmt.Sprintf("   YAML Fix: %s\n", issue.FixYAML))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
