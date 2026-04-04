// Package hooks provides a lifecycle event system for injecting custom logic
// at specific points during execution.
package hooks

import (
	"context"
	"time"
)

// HookEvent represents the type of event that triggers hooks
type HookEvent string

const (
	// Tool lifecycle events
	EventPreToolUse         HookEvent = "PreToolUse"
	EventPostToolUse        HookEvent = "PostToolUse"
	EventPostToolUseFailure HookEvent = "PostToolUseFailure"

	// Session lifecycle events
	EventSessionStart HookEvent = "SessionStart"
	EventSessionEnd   HookEvent = "SessionEnd"

	// Model interaction events
	EventStop        HookEvent = "Stop"
	EventStopFailure HookEvent = "StopFailure"

	// Subagent events
	EventSubagentStart HookEvent = "SubagentStart"
	EventSubagentStop  HookEvent = "SubagentStop"

	// Conversation management events
	EventPreCompact  HookEvent = "PreCompact"
	EventPostCompact HookEvent = "PostCompact"

	// Permission events
	EventPermissionRequest HookEvent = "PermissionRequest"
	EventPermissionDenied  HookEvent = "PermissionDenied"

	// User interaction events
	EventUserPromptSubmit HookEvent = "UserPromptSubmit"
	EventNotification     HookEvent = "Notification"

	// System events
	EventSetup            HookEvent = "Setup"
	EventConfigChange     HookEvent = "ConfigChange"
	EventCwdChanged       HookEvent = "CwdChanged"
	EventFileChanged      HookEvent = "FileChanged"
	EventInstructionsLoad HookEvent = "InstructionsLoaded"

	// Worktree events
	EventWorktreeCreate HookEvent = "WorktreeCreate"
	EventWorktreeRemove HookEvent = "WorktreeRemove"

	// Task events
	EventTaskCreated   HookEvent = "TaskCreated"
	EventTaskCompleted HookEvent = "TaskCompleted"

	// MCP events
	EventElicitation       HookEvent = "Elicitation"
	EventElicitationResult HookEvent = "ElicitationResult"

	// Team events
	EventTeammateIdle HookEvent = "TeammateIdle"
)

// HookType represents the type of hook implementation
type HookType string

const (
	HookTypeCommand  HookType = "command"  // Shell command execution
	HookTypePrompt   HookType = "prompt"   // LLM prompt evaluation
	HookTypeAgent    HookType = "agent"    // Multi-turn LLM verification
	HookTypeHTTP     HookType = "http"     // HTTP POST request
	HookTypeCallback HookType = "callback" // Go callback function
)

// HookSource represents where a hook is configured
type HookSource string

const (
	SourceUserSettings    HookSource = "userSettings"
	SourceProjectSettings HookSource = "projectSettings"
	SourceLocalSettings   HookSource = "localSettings"
	SourcePolicySettings  HookSource = "policySettings"
	SourcePluginHook      HookSource = "pluginHook"
	SourceSessionHook     HookSource = "sessionHook"
	SourceBuiltinHook     HookSource = "builtinHook"
)

// Decision represents a hook's permission decision
type Decision string

const (
	DecisionApprove Decision = "approve"
	DecisionBlock   Decision = "block"
)

// HookMatcher defines matching criteria for hooks
type HookMatcher struct {
	Matcher string `json:"matcher,omitempty"` // Tool name, file pattern, etc.
}

// BaseHook contains common fields for all hook types
type BaseHook struct {
	Type           HookType   `json:"type"`
	If             string     `json:"if,omitempty"`             // Condition filter
	Timeout        int        `json:"timeout,omitempty"`        // Timeout in seconds
	StatusMessage  string     `json:"statusMessage,omitempty"`  // Status message to display
	Once           bool       `json:"once,omitempty"`           // Remove after execution
	Async          bool       `json:"async,omitempty"`          // Execute in background
	AsyncRewake    bool       `json:"asyncRewake,omitempty"`    // Rewake on exit code 2
	Source         HookSource `json:"source,omitempty"`         // Configuration source
	ID             string     `json:"id,omitempty"`             // Unique identifier
}

// CommandHook executes a shell command
type CommandHook struct {
	BaseHook
	Command string `json:"command"`           // Shell command to execute
	Shell   string `json:"shell,omitempty"`   // "bash" or "powershell"
}

// PromptHook sends a prompt to an LLM for evaluation
type PromptHook struct {
	BaseHook
	Prompt string `json:"prompt"`           // LLM prompt ($ARGUMENTS placeholder)
	Model  string `json:"model,omitempty"`  // Specific model to use
}

// AgentHook performs multi-turn LLM verification
type AgentHook struct {
	BaseHook
	Prompt string `json:"prompt"`           // Verification prompt
	Model  string `json:"model,omitempty"`  // Specific model to use
}

// HTTPHook sends an HTTP POST request
type HTTPHook struct {
	BaseHook
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers,omitempty"`
	AllowedEnvVars  []string          `json:"allowedEnvVars,omitempty"`
}

// CallbackHook executes a Go function
type CallbackHook struct {
	BaseHook
	Callback HookCallbackFunc `json:"-"`
	Internal bool             `json:"internal,omitempty"` // Internal hook (not counted in metrics)
}

// HookCallbackFunc is the signature for callback hooks
type HookCallbackFunc func(ctx context.Context, input *HookInput) (*HookOutput, error)

// Hook represents any hook type
type Hook interface {
	GetType() HookType
	GetTimeout() int
	GetStatusMessage() string
	IsOnce() bool
	IsAsync() bool
	GetID() string
}

// Implement Hook interface for all hook types
func (h CommandHook) GetType() HookType      { return h.Type }
func (h CommandHook) GetTimeout() int        { return h.Timeout }
func (h CommandHook) GetStatusMessage() string { return h.StatusMessage }
func (h CommandHook) IsOnce() bool           { return h.Once }
func (h CommandHook) IsAsync() bool          { return h.Async }
func (h CommandHook) GetID() string          { return h.ID }

func (h PromptHook) GetType() HookType       { return h.Type }
func (h PromptHook) GetTimeout() int         { return h.Timeout }
func (h PromptHook) GetStatusMessage() string { return h.StatusMessage }
func (h PromptHook) IsOnce() bool            { return h.Once }
func (h PromptHook) IsAsync() bool           { return h.Async }
func (h PromptHook) GetID() string           { return h.ID }

func (h AgentHook) GetType() HookType        { return h.Type }
func (h AgentHook) GetTimeout() int          { return h.Timeout }
func (h AgentHook) GetStatusMessage() string { return h.StatusMessage }
func (h AgentHook) IsOnce() bool             { return h.Once }
func (h AgentHook) IsAsync() bool            { return h.Async }
func (h AgentHook) GetID() string            { return h.ID }

func (h HTTPHook) GetType() HookType         { return h.Type }
func (h HTTPHook) GetTimeout() int           { return h.Timeout }
func (h HTTPHook) GetStatusMessage() string  { return h.StatusMessage }
func (h HTTPHook) IsOnce() bool              { return h.Once }
func (h HTTPHook) IsAsync() bool             { return h.Async }
func (h HTTPHook) GetID() string             { return h.ID }

func (h CallbackHook) GetType() HookType     { return h.Type }
func (h CallbackHook) GetTimeout() int       { return h.Timeout }
func (h CallbackHook) GetStatusMessage() string { return h.StatusMessage }
func (h CallbackHook) IsOnce() bool          { return h.Once }
func (h CallbackHook) IsAsync() bool         { return h.Async }
func (h CallbackHook) GetID() string         { return h.ID }

// HookInput contains the input data passed to a hook
type HookInput struct {
	Event         HookEvent               `json:"event"`
	ToolName      string                  `json:"toolName,omitempty"`
	ToolUseID     string                  `json:"toolUseId,omitempty"`
	Matcher       string                  `json:"matcher,omitempty"`
	Input         map[string]interface{}  `json:"input,omitempty"`
	Output        interface{}             `json:"output,omitempty"`
	Error         string                  `json:"error,omitempty"`
	SessionID     string                  `json:"sessionId,omitempty"`
	ProjectDir    string                  `json:"projectDir,omitempty"`
	Arguments     string                  `json:"arguments,omitempty"`     // For prompt hooks
	NotificationType string               `json:"notificationType,omitempty"`
	Source        string                  `json:"source,omitempty"`        // Event source
	Reason        string                  `json:"reason,omitempty"`        // End reason
	Trigger       string                  `json:"trigger,omitempty"`       // Compact/setup trigger
	AgentType     string                  `json:"agentType,omitempty"`     // Subagent type
	MCPServerName string                  `json:"mcpServerName,omitempty"`
	OldCwd        string                  `json:"oldCwd,omitempty"`
	NewCwd        string                  `json:"newCwd,omitempty"`
	ChangedFiles  []string                `json:"changedFiles,omitempty"`
	Env           map[string]string       `json:"env,omitempty"`
}

// HookOutput represents the result of a hook execution
type HookOutput struct {
	Continue         bool                   `json:"continue"`                   // Continue execution (default true)
	SuppressOutput   bool                   `json:"suppressOutput,omitempty"`   // Hide output from user
	StopReason       string                 `json:"stopReason,omitempty"`       // Message when continue=false
	Decision         Decision               `json:"decision,omitempty"`         // Permission decision
	Reason           string                 `json:"reason,omitempty"`           // Explanation
	SystemMessage    string                 `json:"systemMessage,omitempty"`    // Inject system message
	UpdatedInput     map[string]interface{} `json:"updatedInput,omitempty"`     // Modified tool input
	AdditionalContext string                `json:"additionalContext,omitempty"` // Extra context
	Async            bool                   `json:"async,omitempty"`            // Mark as async execution
	AsyncTimeout     time.Duration          `json:"asyncTimeout,omitempty"`     // Async timeout
	HookSpecific     map[string]interface{} `json:"hookSpecificOutput,omitempty"`
}

// HookMatcherConfig associates matchers with hooks for an event
type HookMatcherConfig struct {
	Matcher string `json:"matcher,omitempty"`
	Hooks   []Hook `json:"hooks"`
}

// HooksSettings contains all hook configurations
type HooksSettings map[HookEvent][]HookMatcherConfig

// HookExecutionEvent represents a hook lifecycle event
type HookExecutionEvent struct {
	Type      HookEvent `json:"type"`
	HookID    string    `json:"hookId"`
	HookName  string    `json:"hookName"`
	HookEvent HookEvent `json:"hookEvent"`
	Stdout    string    `json:"stdout,omitempty"`
	Stderr    string    `json:"stderr,omitempty"`
	Output    *HookOutput `json:"output,omitempty"`
	ExitCode  int       `json:"exitCode,omitempty"`
	Outcome   string    `json:"outcome,omitempty"` // "success", "blocked", "error"
	Duration  time.Duration `json:"duration,omitempty"`
}

// HookEventHandler handles hook execution events
type HookEventHandler func(event HookExecutionEvent)

// DefaultHookTimeout is the default timeout for hook execution
const DefaultHookTimeout = 60 * time.Second

// GetTimeoutDuration returns the hook's timeout as a Duration
func GetTimeoutDuration(h Hook) time.Duration {
	timeout := h.GetTimeout()
	if timeout <= 0 {
		return DefaultHookTimeout
	}
	return time.Duration(timeout) * time.Second
}
