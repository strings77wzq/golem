package security

// SecurityEvent represents a security-relevant decision that can be observed.
type SecurityEvent string

const (
	EventSQLAllowed  SecurityEvent = "sql_allowed"
	EventSQLDenied   SecurityEvent = "sql_denied"
	EventExecAllowed SecurityEvent = "exec_allowed"
	EventExecDenied  SecurityEvent = "exec_denied"
)

// SecurityEventHandler processes security events.
// Implementations should be non-blocking or bounded.
type SecurityEventHandler func(event SecurityEvent, details map[string]string)
