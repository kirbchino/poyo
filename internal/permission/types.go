// Package permission provides multi-layer permission management for Poyo.
package permission

import (
	"time"
)

// Mode represents a permission mode
type Mode string

const (
	// ModeAsk asks the user for permission each time
	ModeAsk Mode = "ask"
	// ModeAccept automatically allows the operation
	ModeAccept Mode = "accept"
	// ModeAuto automatically classifies based on operation
	ModeAuto Mode = "auto"
	// ModeDeny automatically denies the operation
	ModeDeny Mode = "deny"
)

// Source represents where a permission rule is defined
type Source string

const (
	SourceUser     Source = "userSettings"
	SourceProject  Source = "projectSettings"
	SourceLocal    Source = "localSettings"
	SourcePolicy   Source = "policySettings"
	SourceSession  Source = "sessionRules"
	SourceAuto     Source = "autoClassifier"
)

// Priority represents the priority level of a rule source
type Priority int

const (
	PrioritySession Priority = 100 // Highest - session rules
	PriorityPolicy  Priority = 90  // Enterprise policy
	PriorityLocal   Priority = 70  // Local private settings
	PriorityProject Priority = 50  // Project settings
	PriorityUser    Priority = 30  // User global settings
	PriorityAuto    Priority = 10  // Auto-classifier (lowest)
)

// GetPriority returns the priority for a source
func (s Source) GetPriority() Priority {
	switch s {
	case SourceSession:
		return PrioritySession
	case SourcePolicy:
		return PriorityPolicy
	case SourceLocal:
		return PriorityLocal
	case SourceProject:
		return PriorityProject
	case SourceUser:
		return PriorityUser
	case SourceAuto:
		return PriorityAuto
	default:
		return PriorityUser
	}
}

// Rule represents a permission rule
type Rule struct {
	// Mode is the permission mode
	Mode Mode `json:"mode"`

	// Tool is the tool name or pattern (can be string or array)
	Tool interface{} `json:"tool,omitempty"`

	// Parameters contains additional parameters for the rule
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Reason is a human-readable explanation
	Reason string `json:"reason,omitempty"`

	// Source is where this rule is defined
	Source Source `json:"source,omitempty"`

	// SessionOnly indicates if this rule should only apply for the current session
	SessionOnly bool `json:"sessionOnly,omitempty"`

	// ID is a unique identifier for the rule
	ID string `json:"id,omitempty"`

	// CreatedAt is when the rule was created
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

// GetTools returns the tool names for this rule
func (r *Rule) GetTools() []string {
	switch v := r.Tool.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		tools := make([]string, 0, len(v))
		for _, t := range v {
			if s, ok := t.(string); ok {
				tools = append(tools, s)
			}
		}
		return tools
	default:
		return nil
	}
}

// MatchesTool checks if this rule matches a tool name
func (r *Rule) MatchesTool(toolName string) bool {
	tools := r.GetTools()
	if len(tools) == 0 {
		return true // No tool specified matches all
	}

	for _, t := range tools {
		if t == "*" || t == toolName {
			return true
		}
		// Support glob patterns
		if matchesPattern(t, toolName) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a name matches a glob pattern
func matchesPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}

	// Simple wildcard matching
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		if len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			return true
		}
	}

	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		if len(name) >= len(suffix) && name[len(name)-len(suffix):] == suffix {
			return true
		}
	}

	return pattern == name
}

// RuleSet represents a collection of rules from a single source
type RuleSet struct {
	Source Source
	Rules  []*Rule
}

// Decision represents a permission decision
type Decision struct {
	// Mode is the decided mode
	Mode Mode `json:"mode"`

	// Rule is the rule that made the decision
	Rule *Rule `json:"rule,omitempty"`

	// Reason is the explanation
	Reason string `json:"reason,omitempty"`

	// IsAuto indicates if this was auto-classified
	IsAuto bool `json:"isAuto,omitempty"`

	// Confidence is the confidence level (0-1) for auto-classified decisions
	Confidence float64 `json:"confidence,omitempty"`
}

// ShadowedRule represents a rule that is shadowed by another rule
type ShadowedRule struct {
	// Rule is the shadowed rule
	Rule *Rule `json:"rule"`

	// ShadowedBy is the rule that shadows it
	ShadowedBy *Rule `json:"shadowedBy"`

	// Reason explains why it's shadowed
	Reason string `json:"reason"`
}

// PermissionRequest represents a permission request
type PermissionRequest struct {
	// ToolName is the tool being used
	ToolName string `json:"toolName"`

	// ToolUseID is the unique ID for this tool use
	ToolUseID string `json:"toolUseId"`

	// Input is the tool input
	Input map[string]interface{} `json:"input,omitempty"`

	// Context provides additional context
	Context *PermissionContext `json:"context,omitempty"`
}

// PermissionContext provides context for permission decisions
type PermissionContext struct {
	// WorkingDir is the current working directory
	WorkingDir string `json:"workingDir,omitempty"`

	// ProjectDir is the project root directory
	ProjectDir string `json:"projectDir,omitempty"`

	// IsGitRepo indicates if we're in a git repository
	IsGitRepo bool `json:"isGitRepo,omitempty"`

	// IsTrusted indicates if the current directory is trusted
	IsTrusted bool `json:"isTrusted,omitempty"`

	// SessionID is the current session ID
	SessionID string `json:"sessionId,omitempty"`

	// HasUserInteraction indicates if there was recent user interaction
	HasUserInteraction bool `json:"hasUserInteraction,omitempty"`
}

// PermissionResult represents the result of a permission check
type PermissionResult struct {
	// Decision is the final decision
	Decision *Decision `json:"decision"`

	// RequiresUserInput indicates if user input is needed
	RequiresUserInput bool `json:"requiresUserInput,omitempty"`

	// Allowed indicates if the operation is allowed
	Allowed bool `json:"allowed"`

	// Denied indicates if the operation is denied
	Denied bool `json:"denied,omitempty"`

	// Ask indicates if the user should be asked
	Ask bool `json:"ask,omitempty"`

	// Message is a message to display to the user
	Message string `json:"message,omitempty"`
}

// ClassificationResult represents the result of auto-classification
type ClassificationResult struct {
	// Mode is the classified mode
	Mode Mode `json:"mode"`

	// Confidence is the confidence level (0-1)
	Confidence float64 `json:"confidence"`

	// Reason explains the classification
	Reason string `json:"reason"`

	// Features are the features used for classification
	Features map[string]interface{} `json:"features,omitempty"`
}

// Classifier is the interface for auto-classification
type Classifier interface {
	// Classify classifies a permission request
	Classify(req *PermissionRequest, ctx *PermissionContext) (*ClassificationResult, error)
}
