// Package agent provides sub-agent execution capabilities for Poyo.
package agent

import (
	"context"
	"time"
)

// AgentType defines the type of sub-agent
type AgentType string

const (
	// AgentTypeGeneralPurpose is a general-purpose agent with full tool access
	AgentTypeGeneralPurpose AgentType = "general-purpose"
	// AgentTypeExplore is a fast agent for code exploration (read-only tools)
	AgentTypeExplore AgentType = "Explore"
	// AgentTypePlan is a planning agent for implementation design
	AgentTypePlan AgentType = "Plan"
	// AgentTypeStatuslineSetup is a specialized agent for statusline configuration
	AgentTypeStatuslineSetup AgentType = "statusline-setup"
)

// AgentState represents the state of an agent execution
type AgentState string

const (
	StatePending    AgentState = "pending"
	StateRunning    AgentState = "running"
	StateCompleted  AgentState = "completed"
	StateFailed     AgentState = "failed"
	StateStopped    AgentState = "stopped"
)

// AgentConfig contains configuration for an agent
type AgentConfig struct {
	// Type is the agent type
	Type AgentType `json:"type"`

	// Description is a short description (3-5 words)
	Description string `json:"description"`

	// Prompt is the task for the agent to perform
	Prompt string `json:"prompt"`

	// Resume indicates whether to resume from a previous agent ID
	Resume string `json:"resume,omitempty"`

	// Model overrides the model for this agent
	Model string `json:"model,omitempty"`

	// Isolation mode for execution
	Isolation IsolationMode `json:"isolation,omitempty"`

	// RunInBackground indicates whether to run in background
	RunInBackground bool `json:"run_in_background,omitempty"`
}

// IsolationMode defines how the agent is isolated
type IsolationMode string

const (
	// IsolationNone runs in the current directory
	IsolationNone IsolationMode = ""
	// IsolationWorktree creates a git worktree for isolation
	IsolationWorktree IsolationMode = "worktree"
)

// AgentResult contains the result of an agent execution
type AgentResult struct {
	// AgentID is the unique identifier for this agent run
	AgentID string `json:"agent_id"`

	// State is the current state
	State AgentState `json:"state"`

	// Output is the agent's response
	Output string `json:"output,omitempty"`

	// Error is the error message if failed
	Error string `json:"error,omitempty"`

	// FilesCreated lists files created by the agent
	FilesCreated []string `json:"files_created,omitempty"`

	// FilesModified lists files modified by the agent
	FilesModified []string `json:"files_modified,omitempty"`

	// ToolsUsed lists tools used by the agent
	ToolsUsed []string `json:"tools_used,omitempty"`

	// TokensUsed is the total tokens consumed
	TokensUsed int64 `json:"tokens_used,omitempty"`

	// Duration is the execution duration
	Duration time.Duration `json:"duration,omitempty"`

	// Branch is the git branch (if worktree isolation)
	Branch string `json:"branch,omitempty"`

	// WorktreePath is the worktree path (if worktree isolation)
	WorktreePath string `json:"worktree_path,omitempty"`
}

// AgentContext provides context for agent execution
type AgentContext struct {
	// Context is the standard context
	context.Context

	// WorkingDir is the working directory
	WorkingDir string

	// ConversationID is the parent conversation ID
	ConversationID string

	// AvailableTools is the list of available tools
	AvailableTools []string

	// MaxTurns is the maximum number of turns
	MaxTurns int

	// Timeout is the execution timeout
	Timeout time.Duration
}

// AgentToolAccess defines which tools an agent type can access
type AgentToolAccess struct {
	// AllowedTools is a list of allowed tool patterns
	AllowedTools []string `json:"allowed_tools"`

	// DeniedTools is a list of denied tool patterns
	DeniedTools []string `json:"denied_tools"`

	// ReadOnly indicates if only read-only tools are allowed
	ReadOnly bool `json:"read_only,omitempty"`
}

// DefaultToolAccess returns the default tool access for an agent type
func DefaultToolAccess(agentType AgentType) *AgentToolAccess {
	switch agentType {
	case AgentTypeExplore:
		return &AgentToolAccess{
			AllowedTools: []string{
				"Glob", "Grep", "Read", "Bash",
				"TaskOutput", "Agent",
			},
			DeniedTools: []string{},
			ReadOnly:    true,
		}

	case AgentTypePlan:
		return &AgentToolAccess{
			AllowedTools: []string{
				"Glob", "Grep", "Read", "Bash",
				"TaskOutput", "Agent",
			},
			DeniedTools: []string{},
			ReadOnly:    true,
		}

	case AgentTypeStatuslineSetup:
		return &AgentToolAccess{
			AllowedTools: []string{
				"Read", "Edit", "Write",
			},
			DeniedTools: []string{
				"Bash", "Agent",
			},
			ReadOnly: false,
		}

	default: // general-purpose
		return &AgentToolAccess{
			AllowedTools: []string{"*"},
			DeniedTools:  []string{},
			ReadOnly:     false,
		}
	}
}

// CanUseTool checks if a tool can be used by this agent type
func (a *AgentToolAccess) CanUseTool(toolName string) bool {
	// Check denied tools first
	for _, denied := range a.DeniedTools {
		if denied == "*" || denied == toolName {
			return false
		}
	}

	// Check allowed tools
	for _, allowed := range a.AllowedTools {
		if allowed == "*" || allowed == toolName {
			return true
		}
	}

	return false
}

// AgentRegistry manages active agents
type AgentRegistry struct {
	agents map[string]*AgentInstance
}

// AgentInstance represents a running or completed agent
type AgentInstance struct {
	ID        string
	Type      AgentType
	State     AgentState
	Config    *AgentConfig
	Result    *AgentResult
	StartTime time.Time
	EndTime   time.Time
	Cancel    context.CancelFunc
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*AgentInstance),
	}
}

// Register registers a new agent instance
func (r *AgentRegistry) Register(instance *AgentInstance) {
	r.agents[instance.ID] = instance
}

// Get retrieves an agent instance by ID
func (r *AgentRegistry) Get(id string) (*AgentInstance, bool) {
	instance, ok := r.agents[id]
	return instance, ok
}

// Remove removes an agent instance
func (r *AgentRegistry) Remove(id string) {
	delete(r.agents, id)
}

// List returns all agent instances
func (r *AgentRegistry) List() []*AgentInstance {
	instances := make([]*AgentInstance, 0, len(r.agents))
	for _, instance := range r.agents {
		instances = append(instances, instance)
	}
	return instances
}

// GenerateAgentID generates a unique agent ID
func GenerateAgentID() string {
	return "agent_" + time.Now().Format("20060102_150405")
}
