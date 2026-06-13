package bus

// Role represents the role of a message participant
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
	RoleProgress  Role = "progress"
)

// ProgressType indicates the type of progress event.
type ProgressType string

const (
	ProgressPlanStart  ProgressType = "plan_start"
	ProgressPlanStep   ProgressType = "plan_step"
	ProgressStepStart  ProgressType = "step_start"
	ProgressToolCall   ProgressType = "tool_call"
	ProgressToolResult ProgressType = "tool_result"
	ProgressStepDone   ProgressType = "step_done"
	ProgressStepFailed ProgressType = "step_failed"
	ProgressPlanDone   ProgressType = "plan_done"
)

// InboundMessage represents a message coming into the system
type InboundMessage struct {
	SessionID string
	Content   string
	Role      Role
}

// TokenUsage tracks token consumption for display purposes.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// OutboundMessage represents a message going out from the system
type OutboundMessage struct {
	SessionID    string
	Content      string
	Role         Role
	Done         bool
	TokenDelta   string
	Usage        *TokenUsage
	ProgressType ProgressType
	StepCurrent  int
	StepTotal    int
	ToolName     string
}
