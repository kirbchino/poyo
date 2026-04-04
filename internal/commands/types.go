// Package commands provides slash command functionality for Poyo.
package commands

import (
	"context"
)

// CommandType represents how a command is executed
type CommandType string

const (
	// TypePrompt commands inject their content as a prompt into the conversation
	TypePrompt CommandType = "prompt"
	// TypeLocal commands execute locally and return text output
	TypeLocal CommandType = "local"
	// TypeLocalJSX commands support UI rendering (for TUI/Web interfaces)
	TypeLocalJSX CommandType = "local-jsx"
)

// CommandAvailability controls where a command is available
type CommandAvailability string

const (
	AvailabilityAll     CommandAvailability = "all"
	AvailabilityClaudeAI CommandAvailability = "claude-ai"
	AvailabilityConsole CommandAvailability = "console"
	AvailabilityInternal CommandAvailability = "internal" // Ant/内部用户
)

// Command represents a slash command definition
type Command struct {
	// Name is the primary command name (e.g., "help")
	Name string `json:"name"`

	// Aliases are alternative names (e.g., ["?", "h"])
	Aliases []string `json:"aliases,omitempty"`

	// Description is a short description shown in help
	Description string `json:"description"`

	// LongDescription is detailed help text
	LongDescription string `json:"longDescription,omitempty"`

	// Type determines how the command is executed
	Type CommandType `json:"type"`

	// Prompt is the prompt content for TypePrompt commands
	Prompt string `json:"prompt,omitempty"`

	// Handler is the function for TypeLocal/TypeLocalJSX commands
	Handler CommandHandler `json:"-"`

	// Parameters describes expected arguments
	Parameters string `json:"parameters,omitempty"`

	// Examples shows usage examples
	Examples []string `json:"examples,omitempty"`

	// IsHidden controls visibility in help
	IsHidden bool `json:"isHidden,omitempty"`

	// IsEnabled controls if the command is active
	IsEnabled bool `json:"isEnabled"`

	// Availability restricts where the command is available
	Availability CommandAvailability `json:"availability,omitempty"`

	// RequiresAuth indicates if authentication is required
	RequiresAuth bool `json:"requiresAuth,omitempty"`

	// RequiresProject indicates if a project context is needed
	RequiresProject bool `json:"requiresProject,omitempty"`

	// Category groups related commands
	Category string `json:"category,omitempty"`

	// SortOrder controls display order in help
	SortOrder int `json:"sortOrder,omitempty"`
}

// CommandHandler is the function signature for command execution
type CommandHandler func(ctx context.Context, input *CommandInput) (*CommandOutput, error)

// CommandInput contains the input for command execution
type CommandInput struct {
	// Command is the parsed command
	Command string `json:"command"`

	// Args are the arguments passed to the command
	Args string `json:"args"`

	// SessionID is the current session identifier
	SessionID string `json:"sessionId"`

	// ProjectDir is the current working directory
	ProjectDir string `json:"projectDir"`

	// Environment contains session environment variables
	Environment map[string]string `json:"environment,omitempty"`

	// Metadata can contain additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CommandOutput is the result of command execution
type CommandOutput struct {
	// Output is the text output
	Output string `json:"output"`

	// Prompt is the prompt to inject (for TypePrompt commands)
	Prompt string `json:"prompt,omitempty"`

	// IsError indicates if the output is an error message
	IsError bool `json:"isError,omitempty"`

	// ShouldExit indicates the REPL should exit
	ShouldExit bool `json:"shouldExit,omitempty"`

	// ShouldClear indicates the conversation should be cleared
	ShouldClear bool `json:"shouldClear,omitempty"`

	// ShouldCompact indicates the conversation should be compacted
	ShouldCompact bool `json:"shouldCompact,omitempty"`

	// CompactInstructions are custom instructions for compacting
	CompactInstructions string `json:"compactInstructions,omitempty"`

	// Data contains structured data for UI rendering
	Data interface{} `json:"data,omitempty"`

	// Actions are suggested follow-up actions
	Actions []CommandAction `json:"actions,omitempty"`
}

// CommandAction represents a suggested follow-up action
type CommandAction struct {
	Label   string `json:"label"`
	Command string `json:"command"`
}

// CommandCategory defines command groupings for help display
type CommandCategory string

const (
	CategorySession   CommandCategory = "Session"
	CategoryConfig    CommandCategory = "Configuration"
	CategoryTools     CommandCategory = "Tools & Extensions"
	CategoryGit       CommandCategory = "Git & Code"
	CategoryContext   CommandCategory = "Context & Files"
	CategoryStats     CommandCategory = "Statistics & Export"
	CategoryAccount   CommandCategory = "Account & Auth"
	CategoryDebug     CommandCategory = "Diagnostics"
	CategoryMode      CommandCategory = "Modes & Features"
	CategoryIntegrate CommandCategory = "Integrations"
	CategoryFeedback  CommandCategory = "Feedback & Support"
)

// Match checks if a command name or alias matches the given input
func (c *Command) Match(input string) bool {
	if c.Name == input {
		return true
	}
	for _, alias := range c.Aliases {
		if alias == input {
			return true
		}
	}
	return false
}

// IsAvailable checks if the command is available in the current context
func (c *Command) IsAvailable(availability CommandAvailability, isAuthenticated bool, isInternal bool) bool {
	if !c.IsEnabled {
		return false
	}

	switch c.Availability {
	case AvailabilityAll:
		return true
	case AvailabilityClaudeAI:
		return isAuthenticated
	case AvailabilityConsole:
		return true
	case AvailabilityInternal:
		return isInternal
	default:
		return true
	}
}
